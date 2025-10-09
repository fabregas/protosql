package protosql

import (
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-metrics"
)

var (
	metricsSink *metrics.InmemSink

	useMetrics int64
)

func EnableMetrics(interval, retain time.Duration) {
	metricsSink = metrics.NewInmemSink(interval, retain)

	atomic.StoreInt64(&useMetrics, 1)
}

func LogMetrics(logger Logger) {
	if atomic.LoadInt64(&useMetrics) == 0 {
		return
	}

	snap := metricsSink.Data()
	for _, interval := range snap {
		for name, data := range interval.Gauges {
			logger.Infof("Gauge: %s = %v", name, data.Value)
		}
		for name, data := range interval.Counters {
			logger.Infof("Counter: %s = %d", name, data.Count)
		}
		for name, data := range interval.Samples {
			logger.Infof("Sample: %s = %s", name, data.AggregateSample.String())
		}
	}
}

func addMetricSample(method, q string, val float32) {
	if atomic.LoadInt64(&useMetrics) == 0 {
		return
	}

	metricsSink.AddSampleWithLabels(
		[]string{q},
		val,
		[]metrics.Label{metrics.Label{Name: "method", Value: method}},
	)
}

func addMetricSince(method, q string, t time.Time) {
	addMetricSample(method, q, float32(time.Now().Sub(t)))
}
