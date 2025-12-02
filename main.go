package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Status Enum für Produktionsschritte
const (
	StatusDone       = "DONE"
	StatusInProgress = "IN_PROGRESS"
	StatusFailed     = "FAILED"
)

type ProductionStep struct {
	DeviceID  int    `json:"device_id"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Operator  string `json:"operator"`
}

type metrics struct {
	totalSteps       prometheus.Gauge
	doneCount        prometheus.Gauge
	failedCount      prometheus.Gauge
	inProgressCount  prometheus.Gauge
	completionTime   *prometheus.GaugeVec
	completionHisto  prometheus.Histogram
	failureHisto     prometheus.Histogram
	inProgressHisto  prometheus.Histogram
	doneByHour       *prometheus.GaugeVec
	failedByHour     *prometheus.GaugeVec
	inProgressByHour *prometheus.GaugeVec
}

var steps []ProductionStep
var stepsMutex sync.Mutex
var metricsGlobal *metrics

// --------------------- Hilfsfunktionen ---------------------

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		totalSteps: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Stueckzahl_Produkte",
			Help:      "Gesamtzahl aller Produkte",
		}),
		doneCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Produkte_Abgeschlossen",
			Help:      "Anzahl abgeschlossener Produkte (Status: DONE)",
		}),
		failedCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Produkte_Fehlgeschlagen",
			Help:      "Anzahl fehlgeschlagener Produkte (Status: FAILED)",
		}),
		inProgressCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Produkte_InBearbeitung",
			Help:      "Anzahl in Bearbeitung befindlicher Produkte (Status: IN_PROGRESS)",
		}),
		completionTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Fertigstellung_Zeitstempel",
			Help:      "Unix-Timestamp der fertiggestellten Produkte (nur DONE Status)",
		}, []string{"device_id", "operator"}),
		completionHisto: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Fertigstellung_Zeitstempel_Histogram",
			Help:      "Histogram der Fertigstellungszeiten für DONE Produkte (Stunde des Tages)",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		}),
		failureHisto: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Fehler_Zeitstempel_Histogram",
			Help:      "Histogram der Fehlzeiten für FAILED Produkte (Stunde des Tages)",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		}),
		inProgressHisto: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "InBearbeitung_Zeitstempel_Histogram",
			Help:      "Histogram der Zeiten für IN_PROGRESS Produkte (Stunde des Tages)",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		}),
		doneByHour: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Produkte_Nach_Stunde_Abgeschlossen",
			Help:      "Anzahl der abgeschlossenen Produkte pro Stunde des Tages",
		}, []string{"hour"}),
		failedByHour: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Produkte_Nach_Stunde_Fehlgeschlagen",
			Help:      "Anzahl der fehlgeschlagenen Produkte pro Stunde des Tages",
		}, []string{"hour"}),
		inProgressByHour: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "Produktionsdaten_Mock",
			Name:      "Produkte_Nach_Stunde_InBearbeitung",
			Help:      "Anzahl der IN_PROGRESS Produkte pro Stunde des Tages",
		}, []string{"hour"}),
	}

	reg.MustRegister(
		m.totalSteps, m.doneCount, m.failedCount, m.inProgressCount,
		m.completionTime, m.completionHisto, m.failureHisto, m.inProgressHisto,
		m.doneByHour, m.failedByHour, m.inProgressByHour,
	)

	return m
}

func parseTimestamp(timestamp string) map[string]string {
	result := map[string]string{"date": "", "time": ""}
	formats := []string{time.RFC3339, "2006-01-02T15:04:05"}
	for _, format := range formats {
		t, err := time.Parse(format, timestamp)
		if err == nil {
			result["date"] = t.Format("02/01/2006")
			result["time"] = t.Format("15:04:05")
			return result
		}
	}
	return result
}

func parseHour(timeStr string) int {
	hour := 0
	fmt.Sscanf(timeStr, "%d:", &hour)
	return hour
}

// --------------------- Histogram Helper ---------------------

func observeHistogramForStep(step ProductionStep) {
	timeParsed := parseTimestamp(step.Timestamp)
	if timeParsed["date"] == "" {
		return
	}
	hour := parseHour(timeParsed["time"])
	switch step.Status {
	case StatusDone:
		metricsGlobal.completionHisto.Observe(float64(hour))
	case StatusFailed:
		metricsGlobal.failureHisto.Observe(float64(hour))
	case StatusInProgress:
		metricsGlobal.inProgressHisto.Observe(float64(hour))
	}
}

func rebuildHistograms() {
	metricsGlobal.completionHisto = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "Produktionsdaten_Mock",
		Name:      "Fertigstellung_Zeitstempel_Histogram",
		Help:      "Histogram der Fertigstellungszeiten für DONE Produkte",
		Buckets:   []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	})
	metricsGlobal.failureHisto = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "Produktionsdaten_Mock",
		Name:      "Fehler_Zeitstempel_Histogram",
		Help:      "Histogram der Fehlzeiten für FAILED Produkte",
		Buckets:   []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	})
	metricsGlobal.inProgressHisto = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "Produktionsdaten_Mock",
		Name:      "InBearbeitung_Zeitstempel_Histogram",
		Help:      "Histogram der Zeiten für IN_PROGRESS Produkte",
		Buckets:   []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	})
	prometheus.DefaultRegisterer.Register(metricsGlobal.completionHisto)
	prometheus.DefaultRegisterer.Register(metricsGlobal.failureHisto)
	prometheus.DefaultRegisterer.Register(metricsGlobal.inProgressHisto)

	for _, step := range steps {
		observeHistogramForStep(step)
	}
}

// --------------------- Metrics Update ---------------------

func countByStatusLocked(status string) int {
	count := 0
	for _, step := range steps {
		if step.Status == status {
			count++
		}
	}
	return count
}

func updateMetricsLocked() {
	metricsGlobal.totalSteps.Set(float64(len(steps)))
	metricsGlobal.doneCount.Set(float64(countByStatusLocked(StatusDone)))
	metricsGlobal.failedCount.Set(float64(countByStatusLocked(StatusFailed)))
	metricsGlobal.inProgressCount.Set(float64(countByStatusLocked(StatusInProgress)))

	doneByHourCount := make(map[int]int)
	failedByHourCount := make(map[int]int)
	inProgressByHourCount := make(map[int]int)

	for _, step := range steps {
		timeParsed := parseTimestamp(step.Timestamp)
		if timeParsed["date"] != "" {
			hour := parseHour(timeParsed["time"])
			switch step.Status {
			case StatusDone:
				doneByHourCount[hour]++
			case StatusFailed:
				failedByHourCount[hour]++
			case StatusInProgress:
				inProgressByHourCount[hour]++
			}
		}
	}

	for h := 0; h < 24; h++ {
		hourStr := fmt.Sprintf("%d", h)
		metricsGlobal.doneByHour.WithLabelValues(hourStr).Set(float64(doneByHourCount[h]))
		metricsGlobal.failedByHour.WithLabelValues(hourStr).Set(float64(failedByHourCount[h]))
		metricsGlobal.inProgressByHour.WithLabelValues(hourStr).Set(float64(inProgressByHourCount[h]))
	}
}

// --------------------- HTTP Handler ---------------------

func handleProductionSteps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getProductionsSteps(w, r)
	case http.MethodPost:
		postProductionStep(w, r)
	case http.MethodDelete:
		deleteProductionStep(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getProductionsSteps(w http.ResponseWriter, r *http.Request) {
	stepsMutex.Lock()
	defer stepsMutex.Unlock()
	b, _ := json.Marshal(steps)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func postProductionStep(w http.ResponseWriter, r *http.Request) {
	var newStep ProductionStep
	if err := json.NewDecoder(r.Body).Decode(&newStep); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if newStep.DeviceID == 0 || newStep.Timestamp == "" || newStep.Status == "" || newStep.Operator == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if newStep.Status != StatusDone && newStep.Status != StatusFailed && newStep.Status != StatusInProgress {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	stepsMutex.Lock()
	steps = append(steps, newStep)
	observeHistogramForStep(newStep)
	updateMetricsLocked()
	stepsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Product added successfully", "product": newStep})
}

func deleteProductionStep(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := r.URL.Query().Get("device_id")
	if deviceIDStr == "" {
		http.Error(w, "Missing query parameter: device_id", http.StatusBadRequest)
		return
	}

	var deviceID int
	fmt.Sscanf(deviceIDStr, "%d", &deviceID)

	stepsMutex.Lock()
	defer stepsMutex.Unlock()

	found := false
	for i, step := range steps {
		if step.DeviceID == deviceID {
			steps = append(steps[:i], steps[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	rebuildHistograms()
	updateMetricsLocked()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": fmt.Sprintf("Product with device_id %d deleted successfully", deviceID)})
}

func getCompletionTimes(w http.ResponseWriter, r *http.Request) {
	completedSteps := []map[string]interface{}{}
	for _, step := range steps {
		if step.Status == StatusDone {
			timeParsed := parseTimestamp(step.Timestamp)
			if timeParsed["date"] != "" {
				completedSteps = append(completedSteps, map[string]interface{}{
					"device_id": step.DeviceID,
					"date":      timeParsed["date"],
					"time":      timeParsed["time"],
					"operator":  step.Operator,
				})
			}
		}
	}
	b, _ := json.Marshal(completedSteps)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// --------------------- Main ---------------------

func main() {
	reg := prometheus.NewRegistry()
	metricsGlobal = newMetrics(reg)

	// Initial Histogramme
	stepsMutex.Lock()
	for _, step := range steps {
		observeHistogramForStep(step)
	}
	updateMetricsLocked()
	stepsMutex.Unlock()

	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promHandler.ServeHTTP(w, r)
	})
	http.HandleFunc("/productionSteps", handleProductionSteps)
	http.HandleFunc("/completionTimes", getCompletionTimes)

	fmt.Println("Server listening on :8081")
	http.ListenAndServe(":8081", nil)
}

// --------------------- Initial Data ---------------------

func init() {
	steps = []ProductionStep{
		{123, "2025-11-28T15:29:42", StatusDone, "Randolph"},
		{124, "2025-11-28T12:21:31", StatusFailed, "Amrit"},
		{125, "2025-12-02T11:11:22", StatusDone, "Jörg"},
		{126, "2025-12-01T10:10:10", StatusInProgress, "Daniel"},
		{127, "2025-12-25T09:09:09", StatusDone, "Olga"},
		{128, "2025-11-24T08:08:08", StatusFailed, "Lena"},
		{129, "2025-11-23T07:07:07", StatusDone, "Peer"},
		{130, "2025-11-22T07:07:08", StatusDone, "Chris"},
		{131, "2025-11-21T07:07:09", StatusDone, "Alex"},
		{132, "2025-11-20T07:07:10", StatusDone, "Jordan"},
	}
}
