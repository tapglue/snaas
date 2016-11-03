package metrics

import (
	"fmt"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

// Field names for metric labels.
const (
	FieldComponent = "component"
	FieldMethod    = "method"
	FieldNamespace = "namespace"
	FieldRoute     = "route"
	FieldService   = "service"
	FieldSource    = "source"
	FieldStatus    = "status"
	FieldStore     = "store"
	FieldVersion   = "version"
)

// Common metrics subsystems.
const (
	subsystemErr   = "err"
	subsystemOp    = "op"
	subsystemQueue = "queue"
)

// BucketsQueue are used for Histograms observing queue latencies.
var BucketsQueue = []float64{
	.0005,
	.001,
	.0025,
	.005,
	.01,
	.025,
	.05,
	.1,
	.25,
	.5,
	1,
}

func KeyMetrics(
	namespace string,
	fieldKeys ...string,
) (*kitprometheus.Counter, *kitprometheus.Counter, *prometheus.HistogramVec) {
	errCount := kitprometheus.NewCounterFrom(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemErr,
		Name:      "count",
		Help:      fmt.Sprintf("Number of failed %s operations", namespace),
	}, fieldKeys)

	opCount := kitprometheus.NewCounterFrom(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemOp,
		Name:      "count",
		Help:      fmt.Sprintf("Number of %s operations performed", namespace),
	}, fieldKeys)

	opLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystemOp,
			Name:      "latency_seconds",
			Help:      fmt.Sprintf("Distribution of %s op duration in seconds", namespace),
		},
		fieldKeys,
	)
	prometheus.MustRegister(opLatency)

	return errCount, opCount, opLatency
}
