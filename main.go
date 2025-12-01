package main

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ProductionStep struct {
	DeviceID  int    `json:"device_id"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Operator  string `json:"operator"`
}

type metrics struct {
	step prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		step: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten-Mock",
			Name:      "Stueckzahl Produkte",
			Help:      "Ausgabe von fertigstellten oder nicht fertiggestelllten Produkten",
		}),
	}
	reg.MustRegister(m.step)
	return m
}

var steps []ProductionStep

func init() {
	steps = []ProductionStep{
		{123, "2025-11-28T15:29:42", "DONE", "Randolph"},
		{124, "2025-11-28T12:21:31", "FAILED", "Amrit"},
		{125, "2025-11-27T11:11:22", "DONE", "Jörg"},
	}
}

func main() {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	m.step.Set(float64(len(steps)))

	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	http.Handle("/metrics", promHandler)
	http.HandleFunc("/productionSteps", getProductionsSteps)
	http.ListenAndServe(":8081", nil)
}

func getProductionsSteps(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(steps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
