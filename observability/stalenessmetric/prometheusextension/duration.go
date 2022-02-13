package prometheusextension

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
)

type Duration interface {
	prometheus.Metric
	Start(when time.Time)
	Done(when time.Time)
	Duration() time.Duration
}

var _ Duration = new(durationContainer)

type durationContainer struct {
	desc       *prometheus.Desc
	labelPairs []*dto.LabelPair
	startAt    time.Time
	endAt      time.Time
}

func (d *durationContainer) Start(when time.Time) {
	d.startAt = when
}

func (d *durationContainer) Done(when time.Time) {
	d.endAt = when
}

func (d *durationContainer) Desc() *prometheus.Desc {
	return d.desc
}

func (d *durationContainer) Duration() time.Duration {
	if d.endAt.IsZero() {
		// Ticking
		return time.Since(d.startAt)
	} else {
		return d.endAt.Sub(d.startAt)
	}
}

func (d *durationContainer) Write(out *dto.Metric) error {
	doneLabel := "done"
	doneLabelValue := "0"
	if !d.endAt.IsZero() {
		doneLabelValue = "1"
	}
	out.Label = append(d.labelPairs, &dto.LabelPair{
		Name:  &doneLabel,
		Value: &doneLabelValue,
	})
	out.Gauge = &dto.Gauge{Value: proto.Float64(d.Duration().Seconds())}
	return nil
}

type DurationVec struct {
	*prometheus.MetricVec
	fqName     string
	labelNames []string
}

type DurationOpts prometheus.Opts

func NewDurationVec(opts DurationOpts, labelNames []string) *DurationVec {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		opts.Help,
		labelNames,
		opts.ConstLabels,
	)
	vec := &DurationVec{
		fqName:     prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		labelNames: labelNames,
	}
	vec.MetricVec = prometheus.NewMetricVec(desc, func(lvs ...string) prometheus.Metric {
		if len(lvs) != len(vec.labelNames) {
			panic(fmt.Errorf("%s metric expected label(s) %v but got %d value(s)", vec.fqName, vec.labelNames, len(lvs)))
		}
		result := &durationContainer{
			desc:       desc,
			labelPairs: prometheus.MakeLabelPairs(desc, lvs),
			startAt:    time.Now(),
		}
		return result
	})
	return vec
}

func (d *DurationVec) GetMetricWithLabelValues(lvs ...string) (Duration, error) {
	metric, err := d.MetricVec.GetMetricWithLabelValues(lvs...)
	if metric != nil {
		return metric.(Duration), err
	}
	return nil, err
}

func (d *DurationVec) WithLabelValues(lvs ...string) Duration {
	container, err := d.GetMetricWithLabelValues(lvs...)
	if err != nil {
		panic(err)
	}
	return container
}
