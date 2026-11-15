package media

import (
	"context"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/worker"
)

// 数据保留策略（成熟度审计：media_upload_sessions 未完成会话与对应孤儿
// 对象此前永不清理——应用代码零对象删除调用，媒体只进不出）。
// 模式对齐 notify StartPruneSweeper。
const (
	retentionSweepCadence = time.Hour
	// 未完成上传会话保留 2 天：预签名 URL 短时有效，正常上传在分钟级完成，
	// 2 天覆盖慢网/中断重试窗口，同时把「弃置会话占用存储」的窗口从 7 天压到
	// 2 天（配合 Service.checkPresignQuota 的 presign 配额，共同收敛单账号的
	// 存储放大面；7 天 × 无限弃置是第 81 轮审计的 DoS 残留）。
	staleUploadRetentionDays = 2
	// 单轮最多处理条数：500 条/轮 × 每小时一轮，足以消化配额时代遗留的弃置
	// 会话积压；同时限制单轮时长（每条约 1 次 S3 delete + 1 次 DB delete），
	// 避免单轮长时间占用 sweeping goroutine。
	staleUploadBatchLimit = 500
)

// uploads/ 与 published/ 前缀孤儿对象清理（第 82 轮审计 P2-2；批次 10 泛化到
// published/）。两个前缀共用同一套「列举 → 过滤早于保留期 → 批量引用判定 → 删除
// 无行引用对象」逻辑：
//
//   - uploads/：complete 成功后服务端尽力删除旧上传键，但预签名 PUT URL 在 TTL
//     （mediaPresignTTL，15 分钟）内仍然有效，客户端凭原 URL 重 PUT 即可把 uploads/
//     键写回来——完成态行的 storage_key 已改写为 published/ 键、保留期 GC 只扫未完成
//     行、注销级联只删当前 storage_key，该对象因此永远无人回收。
//   - published/：PrepareComplete 先把对象复制/清洗落盘到不可写发布键
//     published/<userID>/<uploadID>.<ext>，随后 CommitComplete 的条件 UPDATE 才把
//     该键写入 media_upload_sessions。两步之间 DB 失败/进程崩溃时，发布副本没有任何
//     行引用，而会话阶段只删未完成会话的 uploads/ 键，该副本同样永不回收。
//
// 判定：对象早于保留期（staleUploadRetentionDays，2 天）且 DB 无任何
// media_upload_sessions 行引用该 storage_key（任意状态的行都算引用，见
// Repository.ExistingStorageKeys）。published/ 阶段额外把每个发布键的「同 upload 上传
// 键」（uploads/<userID>/<uploadID>.<ext>）并入引用查询：未完成会话行引用的是上传键
// （complete 尚未提交），这类行存在时客户端仍可能重试 complete 并重新发布到该键，
// 此时删发布副本会与在途 complete 竞态（行提交后引用一个已被删的对象）；把兄弟键算作
// 引用即可排除。上传键与发布键由同一 uploadID 唯一派生，行不存在时 complete 永远无法
// 再成功（uploadID 不会重建），因此该判定仍是精确的「无行引用」。
//
// published/ 的安全性：Prepare→Commit 的正常在途窗口只是「一次 DB 条件 UPDATE」，秒级；
// 2 天保留期远大于它。崩溃后客户端重试 complete 会覆盖同一发布键
// （publishedStorageKeyFor 由 userID/uploadID/fileName 确定，幂等），若在保留期内
// 重试成功，该键随 UPDATE 入库、被行引用，即不会被删除；超过保留期后该 upload 会话行
// 通常已被会话阶段 GC 删除（此时 complete 不可能再成功），「旧 + 无行/兄弟键引用」的
// 发布副本只可能是永久孤儿；若 GC 因积压尚未删除该行，兄弟键引用会使其继续保留，直到
// 行被 GC 后下一轮再判定。LastModified 缺失（零值，年龄未知）的对象一律保守跳过并告警。
//
// 单轮有界：每个前缀最多翻 orphanSweepListMaxPages 页（每页 orphanSweepPageSize 条）、
// 最多删 orphanSweepDeleteLimit 条、各自独立的时长预算（见 orphanSweepPlan）；删除失败
// 只告警，下一轮重试（对象仍满足「无行 + 旧」条件）。
//
// 已知限制：uploads/ 阶段按对象键字典序分页、每轮从首页开始（保持批次 9 语义），若前缀
// 对象数超过扫描窗口且窗口内全是仍有引用的键，窗口外对象要等窗口内对象被回收后才会进入
// 扫描；uploads/ 前缀的稳态规模（未完成会话 ≤20/账号 + 2 天内孤儿）远小于窗口上界，当前
// 无需持久化游标。published/ 前缀包含全部完成态媒体、随业务单调增长，固定从首页开始的
// 窗口必然扫不到窗口外，因此该阶段跨轮推进 continuation token（见
// orphanPublishedSweepPlan）；游标只存在进程内、进程重启后从首页重来，属「最终覆盖」
// 而非「每轮全量」。另：删除判定以「本进程所连 DB 的引用」为准，因此同一 bucket 必须
// 只服务同一环境/DB——若多个环境（或多个 DB）共享同一 bucket，published/ 清理会删掉
// 其它环境仍被引用的发布对象，多环境共享存储时不得开启本清理。
const (
	// orphanUploadObjectPrefix 是客户端直传对象键前缀（storageKeyFor 生成
	// uploads/<userID>/<uploadID>.<ext>）。
	orphanUploadObjectPrefix = "uploads/"
	// orphanPublishedObjectPrefix 是 copy-on-complete 的不可写发布键前缀
	// （publishedStorageKeyFor 生成 published/<userID>/<uploadID>.<ext>）。
	orphanPublishedObjectPrefix = "published/"
	// orphanSweepPageSize 是单次 ListObjectsV2 请求的 max-keys：500 条/页让单页响应体
	// 与 DB 批量引用查询（ANY 500 键）都保持轻量。
	orphanSweepPageSize = 500
	// orphanSweepListMaxPages 是单轮、单前缀的翻页上限：20 页 = 每轮每前缀最多扫描
	// 1 万个对象。
	orphanSweepListMaxPages = 20
	// orphanSweepDeleteLimit 是单轮、单前缀的删除上限（与单页上限同值：正常页只有整页
	// 候选都可删时才会触顶，见 sweepOrphanPrefix 的游标说明）：保证单轮时长有界，剩余
	// 对象下一轮继续。
	orphanSweepDeleteLimit = 500
	// orphanUploadSweepTimeout 是 uploads/ 阶段的整体预算（含列举、引用查询、删除）：
	// 超出后剩余对象留待下一轮，避免拖住 sweeper 循环。
	orphanUploadSweepTimeout = 60 * time.Second
	// orphanPublishedSweepTimeout 是 published/ 阶段的整体预算：该阶段绝大多数页只有
	// 列举 + 一次批量引用查询（真正待删除的只有崩溃残留），30s 足够；它与 uploads/
	// 阶段各自独立计时，单轮最坏总时长上界 60+30=90s，远小于 1 小时的清扫节奏。
	orphanPublishedSweepTimeout = 30 * time.Second
)

// orphanSweepPlan 描述单个前缀的单轮清理计划：两个前缀共用同一套列举/判定/删除逻辑，
// 差异只有前缀、时长预算和扫描推进方式（见上方已知限制）。
type orphanSweepPlan struct {
	prefix  string
	timeout time.Duration
	// rotateCursor 为 true 时跨轮推进 continuation token（见 Service.
	// orphanPublishedCursor）；uploads/ 前缀保持批次 9 的「每轮从首页开始」语义。
	rotateCursor bool
	// siblingKey 非空时把每个候选键的「同 upload 兄弟键」一并纳入引用判定
	// （published/ 阶段用，见上方安全性注释）；返回空串表示该键没有兄弟键。
	siblingKey func(storageKey string) string
}

var (
	// orphanUploadSweepPlan：uploads/ 先跑、独立预算，published/ 的扫描量不会挤占它
	// 的页数/删除/时长额度（避免 uploads 饥饿）。
	orphanUploadSweepPlan = orphanSweepPlan{
		prefix:  orphanUploadObjectPrefix,
		timeout: orphanUploadSweepTimeout,
	}
	orphanPublishedSweepPlan = orphanSweepPlan{
		prefix:       orphanPublishedObjectPrefix,
		timeout:      orphanPublishedSweepTimeout,
		rotateCursor: true,
		siblingKey:   uploadsSiblingKeyForPublished,
	}
)

// uploadsSiblingKeyForPublished 返回发布键对应的同 upload 上传键
// （published/<userID>/<uploadID>.<ext> → uploads/<userID>/<uploadID>.<ext>）。
// 非 published/ 前缀返回空串。发布键与上传键由同一 uploadID 唯一派生
// （publishedStorageKeyFor），未完成会话行引用上传键，见上方安全性注释。
func uploadsSiblingKeyForPublished(publishedKey string) string {
	trimmed := strings.TrimPrefix(publishedKey, orphanPublishedObjectPrefix)
	if trimmed == publishedKey {
		return ""
	}
	return orphanUploadObjectPrefix + trimmed
}

var (
	retentionSweepMu      sync.Mutex
	retentionSweepStarted bool
)

// StartRetentionSweeper 启动保留期清理，每轮两个阶段：
//  1. 超期未完成会话：先删其可能已部分上传的存储对象（幂等，404 视为已删），
//     再删会话行；
//  2. uploads/ 与 published/ 前缀孤儿对象：无会话行引用且超过保留期的对象（见
//     sweepOrphanObjects）。
//
// goroutine 绑定 ctx 随优雅停机退出；重复调用 no-op。
//
// ctx/ticker 必须在每次 worker 运行内部创建（见 runRetentionSweeper）：此前在
// 外层用 context.WithCancel 派生 sweepCtx、并在 worker 函数里 defer cancel，
// worker.GoRestartable 在 panic 后重新执行同一闭包时会复用已被 cancel 的
// sweepCtx——重启后的每一轮 sweep 都以 context canceled 静默失败，未完成上传
// 永不回收（第 82 轮审计 P2-4）。
func (s *Service) StartRetentionSweeper(ctx context.Context) {
	retentionSweepMu.Lock()
	defer retentionSweepMu.Unlock()
	if retentionSweepStarted {
		return
	}
	retentionSweepStarted = true
	worker.GoRestartable("media.retention_sweeper", func() {
		// 只有干净退出（ctx 取消）才复位幂等标志：panic 时 GoRestartable 会立即
		// 重启同一闭包，标志必须保持 true，否则并发调用 StartRetentionSweeper
		// 会再起一个 sweeper（重复清扫、goroutine 泄漏）。
		cleanExit := false
		defer func() {
			if !cleanExit {
				return
			}
			retentionSweepMu.Lock()
			retentionSweepStarted = false
			retentionSweepMu.Unlock()
		}()
		s.runRetentionSweeper(ctx)
		cleanExit = true
	})
}

// runRetentionSweeper 是保留期 sweeper 的单次运行周期（GoRestartable panic
// 重启时重新进入本函数）：创建 ticker、启动即先跑一轮，之后按
// retentionSweepCadence 循环直到 ctx 取消。
func (s *Service) runRetentionSweeper(ctx context.Context) {
	ticker := time.NewTicker(retentionSweepCadence)
	defer ticker.Stop()
	s.runRetentionSweep(ctx) // 启动即先跑一轮
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runRetentionSweep(ctx)
		}
	}
}

// runRetentionSweep 执行一轮保留期清理：先回收超期未完成会话（对象+行），再清理
// uploads/ 与 published/ 前缀下已无会话引用的孤儿对象（第 82 轮审计 P2-2，批次 10
// 泛化）。两阶段互相独立：会话阶段失败只影响自身，「行已删对象未删」的残留由孤儿
// 阶段按保留期兜底；孤儿阶段失败也只告警，下一轮重试。
func (s *Service) runRetentionSweep(ctx context.Context) {
	s.sweepStaleUploads(ctx)
	s.sweepOrphanObjects(ctx)
}

func (s *Service) sweepStaleUploads(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cutoff := time.Now().UTC().AddDate(0, 0, -staleUploadRetentionDays)
	stale, err := s.repo.ListStaleUploads(sweepCtx, cutoff, staleUploadBatchLimit)
	if err != nil {
		observability.FromContext(ctx).Warn("media retention sweep list failed", "error", err)
		return
	}
	var deletedObjects, deletedRows int
	for _, item := range stale {
		// 对象可能从未真正上传（presign 后放弃）——404 视为已删，不算失败。
		if err := s.provider.Delete(sweepCtx, item.StorageKey); err != nil {
			observability.FromContext(ctx).Warn("media retention sweep object delete failed",
				"error", err, "upload_id", item.UploadID)
			continue // 对象删除失败则保留行，下一轮重试
		}
		deletedObjects++
		if err := s.repo.DeleteUploadRow(sweepCtx, item.UploadID); err != nil {
			observability.FromContext(ctx).Warn("media retention sweep row delete failed",
				"error", err, "upload_id", item.UploadID)
			continue
		}
		deletedRows++
	}
	if deletedRows > 0 || len(stale) > 0 {
		observability.FromContext(ctx).Info("media retention sweep",
			"stale", len(stale), "objects_deleted", deletedObjects, "rows_deleted", deletedRows)
	}
}

// sweepOrphanObjects 依次清理 uploads/ 与 published/ 两个前缀的孤儿对象。两阶段各自
// 持有独立的页数/删除/时长预算，且按固定顺序执行（uploads/ 先跑）：published/ 前缀
// （全部完成态媒体）比 uploads/ 大几个数量级，若共享单一预算或让 published/ 先跑，
// uploads/ 可能整轮得不到扫描/删除额度（饥饿）；独立预算让两个前缀的推进互不影响。
func (s *Service) sweepOrphanObjects(ctx context.Context) {
	cutoff := time.Now().UTC().AddDate(0, 0, -staleUploadRetentionDays)
	s.sweepOrphanPrefix(ctx, orphanUploadSweepPlan, cutoff)
	s.sweepOrphanPrefix(ctx, orphanPublishedSweepPlan, cutoff)
}

// sweepOrphanPrefix 清理单个前缀下的孤儿对象：逐页列举 → 过滤早于 cutoff（保留期）
// 的对象 → 批量查 DB 引用 → 删除无任何行引用的对象；任一环节失败只告警，下一轮重试。
// 引用状态不可确认时不得删除任何对象（可能误删在途上传/在途发布副本）。
//
// rotateCursor 前缀的游标规则：成功列举后先把游标钉在本页起点，整页处理完再推进到
// NextContinuationToken；这样「引用查询失败 / 单轮预算耗尽 / 删除上限触顶」导致本页
// 提前返回时，下一轮会从本页重扫（已删对象不会重复出现），不会用 nextToken 跳过本页
// 尾部候选造成漏扫。列举失败（可能是保存的 token 已失效）时复位到首页，避免坏 token
// 让该前缀清理永久停摆。
func (s *Service) sweepOrphanPrefix(ctx context.Context, plan orphanSweepPlan, cutoff time.Time) {
	sweepCtx, cancel := context.WithTimeout(ctx, plan.timeout)
	defer cancel()
	logger := observability.FromContext(ctx)
	var scanned, candidates, deleted, failed int
	limitReached := false
	token := ""
	if plan.rotateCursor {
		token = s.orphanPublishedCursor
	}
scanPages:
	for page := 0; page < orphanSweepListMaxPages; page++ {
		pageToken := token
		listing, err := s.provider.ListObjects(sweepCtx, plan.prefix, pageToken, orphanSweepPageSize)
		if err != nil {
			logger.Warn("media retention orphan list failed",
				"error", err, "prefix", plan.prefix, "scanned", scanned, "deleted", deleted)
			if plan.rotateCursor {
				// 保存的 continuation token 可能已失效（对象被批量删除/存储侧翻页
				// 状态过期）：复位到首页，避免坏 token 让该前缀清理永久停摆。
				s.orphanPublishedCursor = ""
			}
			return
		}
		if plan.rotateCursor {
			// 处理本页前先钉住游标（见函数注释的游标规则）。
			s.orphanPublishedCursor = pageToken
		}
		scanned += len(listing.Objects)
		expiredKeys := make([]string, 0, len(listing.Objects))
		for _, object := range listing.Objects {
			if object.Key == "" {
				continue
			}
			if object.LastModified.IsZero() {
				// 无法判断对象年龄：保守跳过（可能是刚上传的对象），告警供排查。
				logger.Warn("media retention orphan object missing last-modified, skipped",
					"prefix", plan.prefix, "storage_key", object.Key)
				continue
			}
			if !object.LastModified.Before(cutoff) {
				continue
			}
			expiredKeys = append(expiredKeys, object.Key)
		}
		candidates += len(expiredKeys)
		if len(expiredKeys) > 0 {
			// published/ 阶段把每个候选发布键的同 upload 上传键一并查引用：未完成
			// 会话行引用上传键，存在时客户端仍可能重试 complete 发布到该键（见上方
			// 安全性注释）。uploads/ 阶段 plan.siblingKey 为 nil，查询集合不变。
			referenceKeys := expiredKeys
			if plan.siblingKey != nil {
				referenceKeys = make([]string, 0, 2*len(expiredKeys))
				for _, storageKey := range expiredKeys {
					referenceKeys = append(referenceKeys, storageKey)
					if sibling := plan.siblingKey(storageKey); sibling != "" {
						referenceKeys = append(referenceKeys, sibling)
					}
				}
			}
			referenced, err := s.repo.ExistingStorageKeys(sweepCtx, referenceKeys)
			if err != nil {
				// 引用状态不可确认时不得删除任何对象：可能误删在途上传的对象。
				logger.Warn("media retention orphan reference check failed",
					"error", err, "prefix", plan.prefix, "candidates", len(expiredKeys))
				return
			}
			for _, storageKey := range expiredKeys {
				if _, ok := referenced[storageKey]; ok {
					continue
				}
				if plan.siblingKey != nil {
					if sibling := plan.siblingKey(storageKey); sibling != "" {
						if _, ok := referenced[sibling]; ok {
							continue
						}
					}
				}
				if err := sweepCtx.Err(); err != nil {
					// 单轮预算耗尽（超时/停机）：剩余对象留给下一轮，避免继续向已取消
					// 的 ctx 发删除请求刷告警；游标已钉在本页起点，本页剩余候选下轮重扫。
					logger.Warn("media retention orphan sweep budget exhausted",
						"error", err, "prefix", plan.prefix, "scanned", scanned,
						"deleted", deleted, "failed", failed)
					return
				}
				if deleted >= orphanSweepDeleteLimit {
					limitReached = true
					break
				}
				if err := s.provider.Delete(sweepCtx, storageKey); err != nil {
					failed++
					logger.Warn("media retention orphan object delete failed",
						"error", err, "prefix", plan.prefix, "storage_key", storageKey)
					continue
				}
				deleted++
			}
		}
		if limitReached {
			break scanPages
		}
		nextToken := ""
		if listing.IsTruncated {
			nextToken = listing.NextContinuationToken
		}
		if plan.rotateCursor {
			s.orphanPublishedCursor = nextToken
		}
		if nextToken == "" {
			break
		}
		token = nextToken
	}
	if scanned > 0 || deleted > 0 {
		logger.Info("media retention orphan sweep",
			"prefix", plan.prefix, "scanned", scanned, "candidates", candidates,
			"objects_deleted", deleted, "failed", failed, "delete_limit_reached", limitReached)
	}
}
