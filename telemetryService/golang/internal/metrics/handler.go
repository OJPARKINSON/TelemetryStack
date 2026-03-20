package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func MetricsHandler() {
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Starting Prometheus metrics server on :9092")
	if err := http.ListenAndServe(":9092", nil); err != nil {
		log.Printf("metrics server failed: %v", err)
	}
}
