package promtest

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func CounterValue(t *testing.T, g prometheus.Gatherer, name string, labels prometheus.Labels) float64 {
	metricFamilies, err := g.Gather()
	if err != nil {
		t.Error(err)
		return 0
	}

	for _, mf := range metricFamilies {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelsEqual(m.GetLabel(), labels) {
				return m.GetCounter().GetValue()
			}
		}
	}

	t.Errorf("metric %q not found", name)
	return 0
}

func labelsEqual(pairs []*dto.LabelPair, labels prometheus.Labels) bool {
	if len(pairs) != len(labels) {
		return false
	}
	for _, pair := range pairs {
		if name, ok := labels[pair.GetName()]; !ok || name != pair.GetValue() {
			return false
		}
	}
	return true
}
