/*
Copyright (c) Facebook, Inc. and its affiliates.
All rights reserved.

This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree.
*/

package exporters

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/prometheus/common/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

const (
	PromoHTTPEndpoint = "/metrics"
	PromoHTTPPort     = 8080
)

// PrometheusExporter handles registering and updating prometheus metrics
type PrometheusExporter struct {
	registeredMetrics map[string]PrometheusMetric
	Registry          prometheus.Registerer
	lock              sync.RWMutex
}

// NewPrometheusExporter create a new PrometheusExporter with own registry
func NewPrometheusExporter() Exporter {
	return &PrometheusExporter{
		registeredMetrics: make(map[string]PrometheusMetric),
		Registry:          prometheus.NewRegistry(),
	}
}

// Submit takes in an ExportSubmission and either registers it to prometheus or
// updates the metric if it is already registered
func (e *PrometheusExporter) Submit(family *dto.MetricFamily, context MetricsContext) error {
	for _, metric := range family.GetMetric() {
		registeredName := makeRegisteredName(metric, family, context.MetricName)
		networkLabels, err := makeNetworkLabels(context.NetworkID, context.GatewayID, metric)
		if err != nil {
			return fmt.Errorf("prometheus submit error %s: %v", registeredName, err)
		}

		e.lock.Lock()
		if registeredMetric, ok := e.registeredMetrics[registeredName]; ok {
			err = registeredMetric.Update(metric, networkLabels)
		} else {
			err = e.registerMetric(metric, family, registeredName, networkLabels)
		}
		if err != nil {
			return fmt.Errorf("prometheus submit error %s: %v", registeredName, err)
		}
		e.lock.Unlock()
	}
	return nil
}

// Start runs the prometheus HTTP exposer in the background
func (e *PrometheusExporter) Start() {
	promoExposer := NewPrometheusHTTPExposer(e)
	go promoExposer.Run(PromoHTTPEndpoint, PromoHTTPPort)
}

// clearRegistry removes all registered metrics from the exporter
func (e *PrometheusExporter) clearRegistry() {
	e.Registry = prometheus.NewRegistry()
	e.registeredMetrics = make(map[string]PrometheusMetric)
}

func makeNetworkLabels(networkID, gatewayID string, metric *dto.Metric) (prometheus.Labels, error) {
	var serviceName = "defaultServiceName"
	for _, label := range metric.GetLabel() {
		if label.GetName() == SERVICE_LABEL_NAME || label.GetName() == "serviceName" {
			serviceName = label.GetValue()
			break
		}
	}

	hostName, err := os.Hostname()
	if err != nil {
		hostName = "defaultHostName"
	}

	return prometheus.Labels{NetworkLabelInstance: networkID, NetworkLabelGateway: gatewayID, NetworkLabelService: serviceName, NetworkLabelHost: hostName}, nil
}

func (e *PrometheusExporter) registerMetric(metric *dto.Metric,
	family *dto.MetricFamily,
	name string,
	networkLabels prometheus.Labels,
) error {
	var newMetric PrometheusMetric
	switch family.GetType() {
	case dto.MetricType_COUNTER:
		newMetric = NewPrometheusCounter(e)
	case dto.MetricType_GAUGE:
		newMetric = NewPrometheusGauge()
	case dto.MetricType_SUMMARY:
		newMetric = NewPrometheusSummary()
	case dto.MetricType_HISTOGRAM:
		newMetric = NewPrometheusHistogram(name)
	default:
		return fmt.Errorf("cannot register unsupported Type: %v", family.GetType())
	}
	return newMetric.Register(metric, name, e, networkLabels)
}

// Takes all labels except service names from metric and adds to the family
// name. Name will be of form: family_name_label0Name_label0Value_label1Name...
func makeRegisteredName(metric *dto.Metric, family *dto.MetricFamily, metricName string) string {
	name := ""
	for _, labelPair := range metric.GetLabel() {
		if labelPair.GetName() == SERVICE_LABEL_NAME || labelPair.GetName() == "serviceName" {
			continue
		}
		name = fmt.Sprintf("%s_%s_%s", name, labelPair.GetName(), labelPair.GetValue())
	}
	registeredName := metricName + name
	return SanitizePrometheusNames(registeredName)
}

// PrometheusHTTPExposer handles exposing a given exporter through http
type PrometheusHTTPExposer struct {
	exporter Exporter
}

// NewPrometheusHTTPExposer returns a new exposer for a given exporter
func NewPrometheusHTTPExposer(exporter *PrometheusExporter) *PrometheusHTTPExposer {
	return &PrometheusHTTPExposer{
		exporter: exporter,
	}
}

// Run blocks as the exporter registry is handled through the given port
// and endpoint
func (e *PrometheusHTTPExposer) Run(endpoint string, port uint64) {
	handler := promhttp.HandlerFor(
		e.exporter.(*PrometheusExporter).Registry.(*prometheus.Registry),
		promhttp.HandlerOpts{
			ErrorLog:      log.NewErrorLogger(),
			ErrorHandling: promhttp.ContinueOnError,
		},
	)
	http.Handle(endpoint, handler)
	http.ListenAndServe(fmt.Sprintf(":%v", port), handler)
}

// Replace any non-alphanumberic or '_' characters with '_'
func SanitizePrometheusNames(name string) string {
	nonPromoChars := regexp.MustCompile("[^a-zA-Z\\d_]")
	nameBytes := []byte(name)
	replaceChar := []byte("_")
	return string(nonPromoChars.ReplaceAll(nameBytes, replaceChar))
}
