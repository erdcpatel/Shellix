package httpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"shellix/internal/core"
)

// Execute sends an HTTP request with fine-grained network tracing
func (c *Client) Execute(opts RequestOptions) (*ResponseSummary, error) {
	if opts.Method == "" {
		opts.Method = "GET"
	}
	opts.Method = strings.ToUpper(opts.Method)

	if !strings.HasPrefix(opts.URL, "http://") && !strings.HasPrefix(opts.URL, "https://") {
		opts.URL = "http://" + opts.URL
	}

	var reqBody io.Reader
	if len(opts.Body) > 0 {
		reqBody = bytes.NewReader(opts.Body)
	}

	req, err := http.NewRequest(opts.Method, opts.URL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply headers
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}
	if opts.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.BearerToken)
		core.GlobalSanitizer.RegisterSecret(opts.BearerToken)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Shellix-HTTP-Client/0.1.0")
	}

	var metrics TimingMetrics
	var dnsStart, connStart, tlsStart, reqStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			metrics.DNSDuration = time.Since(dnsStart)
		},
		ConnectStart: func(network, addr string) {
			connStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			metrics.ConnDuration = time.Since(connStart)
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			metrics.TLSDuration = time.Since(tlsStart)
		},
		GotFirstResponseByte: func() {
			metrics.TTFB = time.Since(reqStart)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	reqStart = time.Now()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	metrics.TotalTime = time.Since(reqStart)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	isJSON := false
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") || json.Valid(bodyBytes) {
		isJSON = true
	}

	return &ResponseSummary{
		StatusCode:    resp.StatusCode,
		StatusText:    resp.Status,
		Headers:       resp.Header,
		Body:          bodyBytes,
		ContentLength: int64(len(bodyBytes)),
		Timing:        metrics,
		IsJSON:        isJSON,
	}, nil
}
