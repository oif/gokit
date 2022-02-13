package stalenessmetric

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Gauge interface {
	Holder
	WithLabelValues(lvs ...string) prometheus.Gauge
}

var _ Gauge = new(stalenessGauge)

type stalenessGauge struct {
	*holder
}

func NewGauge(raw *prometheus.GaugeVec, expiration time.Duration, registry prometheus.Registerer) *stalenessGauge {
	counter := &stalenessGauge{
		holder: newHolder(raw, expiration),
	}
	mustRegister(registry, raw)
	return counter
}

func (g *stalenessGauge) WithLabelValues(lvs ...string) prometheus.Gauge {
	return g.holder.TryLabelValues(lvs...).(prometheus.Gauge)
}
