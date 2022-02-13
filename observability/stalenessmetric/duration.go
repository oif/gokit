package stalenessmetric

import (
	"time"

	"github.com/oif/gokit/observability/stalenessmetric/prometheusextension"

	"github.com/prometheus/client_golang/prometheus"
)

type Duration interface {
	Holder
	WithLabelValues(lvs ...string) prometheusextension.Duration
}

type stalenessDuration struct {
	*holder
}

func NewDuration(raw *prometheusextension.DurationVec, expiration time.Duration, registry prometheus.Registerer) *stalenessDuration {
	duration := &stalenessDuration{
		holder: newHolder(raw, expiration),
	}
	mustRegister(registry, raw)
	return duration
}

func (d *stalenessDuration) WithLabelValues(lvs ...string) prometheusextension.Duration {
	return d.holder.TryLabelValues(lvs...).(prometheusextension.Duration)
}
