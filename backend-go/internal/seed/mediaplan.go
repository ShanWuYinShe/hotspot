package seed

import (
	"fmt"
	"strings"
	"time"
)

// mediaplan.go — 媒体写入计划：把素材映射为「上传会话 + 对象存储对象 + file_id」。
//
// 每个媒体位对应一条 media_upload_sessions 行（status=completed）和一个
// SeaweedFS 对象（key = uploads/<user_id>/<upload_id>.jpg）。file_id/storage_key
// 规则与 media 模块完全一致（见 naming.go）。
//
// 素材复用策略：12 张 moment_*.jpg 在 22 个瞬间图位上轮换（upload_id 不同
// 即 file_id 不同，media 契约也允许同一内容多次上传得到不同 file_id）；
// 聊天/群聊图片同样复用瞬间素材（分类为 chat_image）。

// mediaObject 描述一次「已完成上传」的媒体会话 + 待 PUT 的对象。
type mediaObject struct {
	UploadID    string
	UserID      string // 上传者（avatar/group_avatar 为资料所有者；moment_image 为作者；chat_image 为发送者）
	Category    string // media.MediaCategory 的字符串值
	FileName    string
	StorageKey  string
	FileID      string
	MimeType    string
	FileSize    int
	MD5         string
	Width       int
	Height      int
	CreatedAt   time.Time
	CompletedAt time.Time
	AssetName   string // 素材文件名（上传时从 //go:embed 读取）
}

// mediaBundle 是媒体计划的输出：对象列表 + 供其它表引用的 file_id 查找表。
type mediaBundle struct {
	objects            []mediaObject
	userAvatarFileID   map[string]string
	groupAvatarFileID  map[string]string
	momentImageFileIDs map[string][]string
	chatImageFileIDs   map[string]string
}

// momentImageAssetPool 是瞬间图/聊天图可轮换使用的素材（12 张）。
func momentImageAssetPool() []string {
	pool := make([]string, 0, 12)
	for i := 1; i <= 12; i++ {
		pool = append(pool, "moment_"+twoDigits(i)+".jpg")
	}
	return pool
}

// chatImageKeys 是聊天图片的固定媒体键（顺序即创建顺序，保证可复现）。
var chatImageKeys = []string{"chat_img_mika", "chat_img_sophia", "gmsg_img_1"}

// chatImageAssets 指定聊天图片键 → 素材文件。
func chatImageAssets(key string) string {
	switch key {
	case "chat_img_mika":
		return "moment_10.jpg"
	case "chat_img_sophia":
		return "moment_06.jpg"
	case "gmsg_img_1":
		return "moment_04.jpg"
	default:
		return "moment_01.jpg"
	}
}

// buildMediaPlan 构建全部媒体对象与 file_id 查找表。
//
// 返回错误只发生在数据规格自相矛盾时（引用了不存在的用户/素材、聊天图片
// 键找不到对应消息），属于开发期错误，执行 seed 时立即失败。
func buildMediaPlan(baseTime time.Time, data seedData, assets map[string]seedAsset) (*mediaBundle, error) {
	bundle := &mediaBundle{
		userAvatarFileID:   make(map[string]string, len(data.users)),
		groupAvatarFileID:  make(map[string]string, len(data.groups)),
		momentImageFileIDs: make(map[string][]string, len(data.moments)),
		chatImageFileIDs:   make(map[string]string, len(chatImageKeys)),
	}
	userByID := make(map[string]userSpec, len(data.users))
	for _, user := range data.users {
		userByID[user.UserID] = user
	}

	add := func(obj mediaObject) error {
		asset, ok := assets[obj.AssetName]
		if !ok {
			return fmt.Errorf("媒体 %s 引用了不存在的素材 %s", obj.UploadID, obj.AssetName)
		}
		obj.MimeType = seedJPEGMimeType
		obj.FileSize = asset.Size
		obj.MD5 = asset.MD5
		obj.Width = asset.Width
		obj.Height = asset.Height
		bundle.objects = append(bundle.objects, obj)
		return nil
	}

	// 1) 用户头像。
	for _, user := range data.users {
		if user.AvatarAsset == "" {
			continue
		}
		uploadID := uploadIDPrefix + "avatar_" + strings.TrimPrefix(user.UserID, userIDPrefix)
		fileID := buildFileID(user.UserID, uploadID)
		if err := add(mediaObject{
			UploadID:    uploadID,
			UserID:      user.UserID,
			Category:    "avatar",
			FileName:    user.AvatarAsset,
			AssetName:   user.AvatarAsset,
			StorageKey:  storageKeyFor(user.UserID, uploadID, user.AvatarAsset),
			FileID:      fileID,
			CreatedAt:   baseTime.Add(-30 * oneDay),
			CompletedAt: baseTime.Add(-30*oneDay + time.Minute),
		}); err != nil {
			return nil, err
		}
		bundle.userAvatarFileID[user.UserID] = fileID
	}

	// 2) 群头像（上传者为群主，category=group_avatar）。
	for _, group := range data.groups {
		if group.AvatarAsset == "" {
			continue
		}
		if _, ok := userByID[group.OwnerID]; !ok {
			return nil, fmt.Errorf("群 %s 的群主 %s 不在用户规格中", group.GroupID, group.OwnerID)
		}
		uploadID := uploadIDPrefix + "group_" + strings.TrimPrefix(group.GroupID, groupIDPrefix)
		fileID := buildFileID(group.OwnerID, uploadID)
		created := baseTime.Add(-group.CreatedAgo)
		if err := add(mediaObject{
			UploadID:    uploadID,
			UserID:      group.OwnerID,
			Category:    "group_avatar",
			FileName:    group.AvatarAsset,
			AssetName:   group.AvatarAsset,
			StorageKey:  storageKeyFor(group.OwnerID, uploadID, group.AvatarAsset),
			FileID:      fileID,
			CreatedAt:   created,
			CompletedAt: created.Add(2 * time.Minute),
		}); err != nil {
			return nil, err
		}
		bundle.groupAvatarFileID[group.GroupID] = fileID
	}

	// 3) 瞬间图：按瞬间顺序与图片位轮换素材。
	pool := momentImageAssetPool()
	slot := 0
	for _, moment := range data.moments {
		if moment.ImageCount <= 0 {
			continue
		}
		if _, ok := userByID[moment.AuthorID]; !ok {
			return nil, fmt.Errorf("瞬间 %s 的作者 %s 不在用户规格中", moment.MomentID, moment.AuthorID)
		}
		fileIDs := make([]string, 0, moment.ImageCount)
		for i := 0; i < moment.ImageCount; i++ {
			assetName := pool[slot%len(pool)]
			slot++
			momentSlug := strings.TrimPrefix(moment.MomentID, momentIDPrefix)
			uploadID := fmt.Sprintf("%s%s_img%d", uploadIDPrefix, momentSlug, i+1)
			fileID := buildFileID(moment.AuthorID, uploadID)
			created := baseTime.Add(-moment.CreatedAgo - 5*time.Minute)
			if err := add(mediaObject{
				UploadID:    uploadID,
				UserID:      moment.AuthorID,
				Category:    "moment_image",
				FileName:    assetName,
				AssetName:   assetName,
				StorageKey:  storageKeyFor(moment.AuthorID, uploadID, assetName),
				FileID:      fileID,
				CreatedAt:   created,
				CompletedAt: created.Add(time.Minute),
			}); err != nil {
				return nil, err
			}
			fileIDs = append(fileIDs, fileID)
		}
		bundle.momentImageFileIDs[moment.MomentID] = fileIDs
	}

	// 4) 聊天图片：按媒体键在群消息/私聊消息里反查发送者与发送时间。
	for _, key := range chatImageKeys {
		ownerID, sentAt, found, err := findChatMediaMessage(baseTime, data, key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("聊天媒体键 %s 未在群消息/私聊消息规格中找到", key)
		}
		assetName := chatImageAssets(key)
		uploadID := uploadIDPrefix + key
		fileID := buildFileID(ownerID, uploadID)
		created := sentAt.Add(-5 * time.Minute)
		if err := add(mediaObject{
			UploadID:    uploadID,
			UserID:      ownerID,
			Category:    "chat_image",
			FileName:    assetName,
			AssetName:   assetName,
			StorageKey:  storageKeyFor(ownerID, uploadID, assetName),
			FileID:      fileID,
			CreatedAt:   created,
			CompletedAt: created.Add(time.Minute),
		}); err != nil {
			return nil, err
		}
		bundle.chatImageFileIDs[key] = fileID
	}

	return bundle, nil
}

// findChatMediaMessage 在群消息与私聊消息规格中反查媒体键对应的发送者与时间。
func findChatMediaMessage(baseTime time.Time, data seedData, key string) (ownerID string, sentAt time.Time, found bool, err error) {
	for _, message := range data.groupMessages {
		if message.MediaKey != key {
			continue
		}
		if message.SenderID == "" {
			return "", time.Time{}, false, fmt.Errorf("群消息 %s 引用聊天媒体但无发送者", message.MessageID)
		}
		return message.SenderID, baseTime.Add(-message.SentAgo), true, nil
	}
	for _, chat := range data.chats {
		for _, message := range chat.Messages {
			if message.MediaKey != key {
				continue
			}
			return message.SenderID, baseTime.Add(-message.SentAgo), true, nil
		}
	}
	return "", time.Time{}, false, nil
}
