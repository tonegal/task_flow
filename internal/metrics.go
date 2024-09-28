package internal

import (
	"fmt"
	"net/http"

	"example.com/task_flow/config"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartMetricsServer starts the Prometheus metrics server on a background goroutine
func StartMetricsServer(port string) {
	cfg := config.LoadConfig()

	go func() {
		fmt.Printf("Starting metrics server on port %s...\n", port)
		http.Handle(cfg.Prometheus.Metrics_Endpoint, promhttp.Handler())
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			panic(err)
		}
	}()
}
