package prometheusextension

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	durationVec := NewDurationVec(DurationOpts{
		Namespace: "testing",
		Name:      "try",
	}, []string{"abc"})
	durationVec.WithLabelValues("123").Start(time.Now().Add(-time.Second))
	assert.Equal(t, 1, int(durationVec.WithLabelValues("123").Duration().Seconds()))
	time.Sleep(time.Second)
	assert.Equal(t, 2, int(durationVec.WithLabelValues("123").Duration().Seconds()))
	durationVec.WithLabelValues("123").Done(time.Now())
	time.Sleep(time.Second)
	assert.Equal(t, 2, int(durationVec.WithLabelValues("123").Duration().Seconds()))
}
