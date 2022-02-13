package stalenessmetric

import (
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/oif/gokit/wait"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricVec interface {
	prometheus.Collector

	DeleteLabelValues(lvs ...string) bool
}

const (
	labelKeySeparator = "^#"
)

type Holder interface {
	IsExpired(labels ...string) bool
}

type holder struct {
	metric     MetricVec
	expiry     sync.Map
	expiration time.Duration
}

func newHolder(metric MetricVec, expiration time.Duration) *holder {
	h := &holder{
		metric:     metric,
		expiration: expiration,
	}
	go wait.Keep(h.cleaner, time.Minute, false, make(chan struct{}))
	return h
}

func (h *holder) cleaner() {
	h.expiry.Range(func(key, value interface{}) bool {
		expiry := value.(time.Time)
		if expiry.Before(time.Now()) {
			keys := strings.Split(key.(string), labelKeySeparator)
			h.metric.DeleteLabelValues(keys...)
			h.expiry.Delete(key)
		}
		return true
	})
}

func (h *holder) TryLabelValues(lvs ...string) interface{} {
	h.expiry.Store(strings.Join(lvs, labelKeySeparator), time.Now().Add(h.expiration))
	args := make([]reflect.Value, len(lvs))
	for i, labelValue := range lvs {
		args[i] = reflect.ValueOf(labelValue)
	}
	returnValues := reflect.ValueOf(h.metric).MethodByName("WithLabelValues").Call(args)
	// Should returns a single value, so allowed to panic here(nil is a nonsense return value)
	return returnValues[0].Interface()
}

func (h *holder) IsExpired(labels ...string) bool {
	val, ok := h.expiry.Load(strings.Join(labels, labelKeySeparator))
	if !ok {
		return true
	}
	return val.(time.Time).Before(time.Now())
}
