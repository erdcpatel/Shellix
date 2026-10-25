package httpclient

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// TimingMetrics breaks down HTTP network latency
type TimingMetrics struct {
	DNSStart     time.Time     `json:"dns_start"`
	DNSDuration  time.Duration `json:"dns_duration"`
	ConnStart    time.Time     `json:"conn_start"`
	ConnDuration time.Duration `json:"conn_duration"`
	TLSStart     time.Time     `json:"tls_start"`
	TLSDuration  time.Duration `json:"tls_duration"`
	TTFB         time.Duration `json:"ttfb"` // Time to First Byte
	TotalTime    time.Duration `json:"total_time"`
}

// RequestOptions specifies HTTP request parameters
type RequestOptions struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	BearerToken string            `json:"bearer_token"`
	Timeout     time.Duration     `json:"timeout"`
	InsecureTLS bool              `json:"insecure_tls"`
}

// ResponseSummary captures HTTP response details and timing
type ResponseSummary struct {
	StatusCode    int                 `json:"status_code"`
	StatusText    string              `json:"status_text"`
	Headers       map[string][]string `json:"headers"`
	Body          []byte              `json:"body"`
	ContentLength int64               `json:"content_length"`
	Timing        TimingMetrics       `json:"timing"`
	IsJSON        bool                `json:"is_json"`
}

// DiffResult compares two HTTP responses or JSON structures
type DiffResult struct {
	URL1          string          `json:"url1"`
	URL2          string          `json:"url2"`
	StatusMatch   bool            `json:"status_match"`
	Status1       int             `json:"status1"`
	Status2       int             `json:"status2"`
	Latency1      time.Duration   `json:"latency1"`
	Latency2      time.Duration   `json:"latency2"`
	LatencyDelta  time.Duration   `json:"latency_delta"`
	ContentMatch  bool            `json:"content_match"`
	AddedKeys     []string        `json:"added_keys"`
	RemovedKeys   []string        `json:"removed_keys"`
	ChangedValues []ValueMismatch `json:"changed_values"`
}

// ValueMismatch records a modified JSON property
type ValueMismatch struct {
	Path   string      `json:"path"`
	Value1 interface{} `json:"value1"`
	Value2 interface{} `json:"value2"`
}

// Client wraps an http.Client with tracing capabilities
type Client struct {
	httpClient *http.Client
}

// NewClient initializes a HTTP client with custom timeout and TLS settings
func NewClient(timeout time.Duration, insecureTLS bool) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureTLS,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects to prevent redirect loop")
				}
				return nil
			},
		},
	}
}
