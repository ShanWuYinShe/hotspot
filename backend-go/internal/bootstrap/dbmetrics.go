package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"hotspot/backend-go/internal/platform/persistence"
)

// dbmetrics.go — pgxpool 连接池指标采集器（装配见 NewApp）。
//
// 连接池耗尽是最常见的慢性故障来源（表现为请求延迟抬升后超时），
// empty_acquire_total（acquire 时池为空的累计次数）持续增长即池吃紧信号，
// 配合 acquired/idle/total 三组即时值是排查第一入口。

type dbPoolCollector struct {
	db           persistence.DB
	totalConns   *prometheus.Desc
	idleConns    *prometheus.Desc
	acquiring    *prometheus.Desc
	emptyAcquire *prometheus.Desc
}

// NewDBPoolCollector 暴露 pgxpool 实时统计。db 的底层实现不是 PostgresDB
// （内存后端/测试桩）时，各指标恒为 0。
func NewDBPoolCollector(db persistence.DB) prometheus.Collector {
	return dbPoolCollector{
		db: db,
		totalConns: prometheus.NewDesc(
			"pgxpool_total_conns", "连接池总连接数（含空闲）", nil, nil),
		idleConns: prometheus.NewDesc(
			"pgxpool_idle_conns", "连接池空闲连接数", nil, nil),
		acquiring: prometheus.NewDesc(
			"pgxpool_acquiring_conns", "正在执行查询的连接数", nil, nil),
		emptyAcquire: prometheus.NewDesc(
			"pgxpool_empty_acquire_total", "acquire 时池为空的累计次数（持续增长即池吃紧）", nil, nil),
	}
}

func (c dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalConns
	ch <- c.idleConns
	ch <- c.acquiring
	ch <- c.emptyAcquire
}

func (c dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	var pool *pgxpool.Pool
	if pg, ok := c.db.(*persistence.PostgresDB); ok {
		pool = pg.Pool
	}
	if pool == nil {
		// 非 pgx 后端（内存/测试桩）：报零值而非不采集，保持指标序列连续。
		ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.acquiring, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.emptyAcquire, prometheus.CounterValue, 0)
		return
	}
	stat := pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stat.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stat.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.acquiring, prometheus.GaugeValue, float64(stat.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquire, prometheus.CounterValue, float64(stat.EmptyAcquireCount()))
}
