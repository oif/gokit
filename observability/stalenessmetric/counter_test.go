package stalenessmetric_test

import (
	"testing"
	"time"

	"github.com/oif/gokit/observability/stalenessmetric"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestStalenessCounter(t *testing.T) {
	counter := stalenessmetric.NewCounter(prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "testing",
		}, []string{"kind", "kind1"}),
		time.Second, nil)
	labels := []string{"1", "2"}
	counter.WithLabelValues(labels...).Add(1)
	assert.False(t, counter.IsExpired(labels...))
	time.Sleep(time.Second)
	assert.True(t, counter.IsExpired(labels...))
}
