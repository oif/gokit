package stalenessmetric

import "github.com/prometheus/client_golang/prometheus"

func mustRegister(registry prometheus.Registerer, cs ...prometheus.Collector) {
	if registry == nil {
		prometheus.MustRegister(cs...)
	} else {
		registry.MustRegister(cs...)
	}
}
