package stalenessmetric

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Counter interface {
	Holder
	WithLabelValues(lvs ...string) prometheus.Counter
}

var _ Counter = new(stalenessCounter)

type stalenessCounter struct {
	*holder
}

func NewCounter(rawCounter *prometheus.CounterVec, expiration time.Duration, registry prometheus.Registerer) *stalenessCounter {
	counter := &stalenessCounter{
		holder: newHolder(rawCounter, expiration),
	}
	mustRegister(registry, rawCounter)
	return counter
}

func (c *stalenessCounter) WithLabelValues(lvs ...string) prometheus.Counter {
	return c.holder.TryLabelValues(lvs...).(prometheus.Counter)
}
