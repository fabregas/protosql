package protosql

import (
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-metrics"
)

var (
	metricsSink *metrics.InmemSink

	useMetrics atomic.Bool
)

func EnableMetrics() {
	metricsSink = metrics.NewInmemSink(60*time.Second, 10*time.Minute)

	useMetrics.Store(true)
}

func LogMetrics(logger Logger) {
	snap := metricsSink.Data()
	for _, interval := range snap {
		for name, data := range interval.Gauges {
			logger.Infof("Gauge: %s = %v", name, data.Value)
		}
		for name, data := range interval.Counters {
			logger.Infof("Counter: %s = %d", name, data.Count)
		}
		for name, data := range interval.Samples {
			logger.Infof("Sample: %s = count=%d, mean=%.2f",
				name, data.Count, data.Mean)
		}
	}
}

func addMetricSample(method, q string, val float32) {
	if !useMetrics.Load() {
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
