package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog/log"
)

// Parser names.
const (
	ParserNginx   = "nginx"
	ParserTraefik = "traefik"
	ParserHAProxy = "haproxy"
)

// Metric names.
const (
	MetricRequestDuration     = "request_duration"
	MetricRequests            = "requests"
	MetricRequestErrors       = "request_errors"
	MetricRequestClientErrors = "request_client_errors"
)

// Metric represents a metric object.
type Metric interface {
	IngressName() string
	ServiceName() string
}

// Counter represents a counter metric.
type Counter struct {
	Name    string
	Ingress string
	Service string
	Value   uint64
}

// CounterFromMetric returns a counter metric from a prometheus
// metric.
func CounterFromMetric(m *dto.Metric) *Counter {
	c := m.Counter
	if c == nil || c.GetValue() == 0 {
		return nil
	}

	return &Counter{
		Value: uint64(c.GetValue()),
	}
}

// IngressName returns the metric ingress name.
func (c Counter) IngressName() string {
	return c.Ingress
}

// ServiceName returns the metric service name.
func (c Counter) ServiceName() string {
	return c.Service
}

// Histogram represents a histogram metric.
type Histogram struct {
	Name     string
	Relative bool
	Ingress  string
	Service  string
	Sum      float64
	Count    uint64
}

// HistogramFromMetric returns a histogram metric from a prometheus
// metric.
func HistogramFromMetric(m *dto.Metric) *Histogram {
	hist := m.Histogram
	if hist == nil || hist.GetSampleCount() == 0 {
		return nil
	}

	return &Histogram{
		Sum:   hist.GetSampleSum(),
		Count: hist.GetSampleCount(),
	}
}

// IngressName returns the metric ingress name.
func (h Histogram) IngressName() string {
	return h.Ingress
}

// ServiceName returns the metric service name.
func (h Histogram) ServiceName() string {
	return h.Service
}

// Parser represents a platform-specific metrics parser.
type Parser interface {
	Parse(m *dto.MetricFamily, svcs map[string][]string) []Metric
}

// Scraper scrapes metrics from Prometheus.
type Scraper struct {
	client *http.Client

	nginxParser   NginxParser
	traefikParser TraefikParser
	haproxyParser HAProxyParser
}

// NewScraper returns a scraper instance with parser p.
func NewScraper(c *http.Client) *Scraper {
	return &Scraper{
		client: c,
	}
}

// Scrape returns metrics scraped from all targets.
func (s *Scraper) Scrape(ctx context.Context, parser string, targets []string, ingressSvcs map[string][]string) ([]Metric, error) {
	// This is a naive approach and should be dealt with
	// as an iterator later to control the amount of RAM
	// used while scraping many targets with many services.
	// e.g. 100 pods * 4000 services * 4 metrics = bad news bears (1.6 million)

	var p Parser
	switch parser {
	case ParserNginx:
		p = s.nginxParser
	case ParserTraefik:
		p = s.traefikParser
	case ParserHAProxy:
		p = s.haproxyParser
	default:
		return nil, fmt.Errorf("unvalid parser %q", parser)
	}

	var m []Metric

	// Flip the relationship to make it quicker to look up.
	svcIngresses := map[string][]string{}
	for ingr, svcs := range ingressSvcs {
		for _, svc := range svcs {
			ingrs := svcIngresses[svc]
			ingrs = append(ingrs, ingr)
			svcIngresses[svc] = ingrs
		}
	}

	for _, u := range targets {
		raw, err := s.scrapeMetrics(ctx, u)
		if err != nil {
			log.Error().Err(err).Str("target", u).Msg("Unable to get metrics from target")
			continue
		}

		for _, v := range raw {
			m = append(m, p.Parse(v, svcIngresses)...)
		}
	}

	return m, nil
}

func (s *Scraper) scrapeMetrics(ctx context.Context, target string) ([]*dto.MetricFamily, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("scraper: unexpected status code from target url " + target)
	}

	var m []*dto.MetricFamily
	dec := expfmt.NewDecoder(resp.Body, expfmt.ResponseFormat(resp.Header))
	for {
		var fam dto.MetricFamily
		err = dec.Decode(&fam)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return m, nil
			}

			return nil, err
		}

		m = append(m, &fam)
	}
}
