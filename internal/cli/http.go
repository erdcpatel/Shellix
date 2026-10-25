package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"shellix/internal/httpclient"
	"shellix/internal/jsonview"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewHTTPCmd constructs the HTTP & API client CLI
func NewHTTPCmd(app *AppContext) *cobra.Command {
	var headers []string
	var data string
	var bearer string
	var timeoutStr string
	var insecure bool
	var verbose bool
	var outputFile string

	// Assertions for automated API health checks
	var assertStatus int
	var assertJSON string
	var assertMaxLatency string

	httpCmd := &cobra.Command{
		Use:     "http",
		Aliases: []string{"api", "curl"},
		Short:   "High-speed HTTP & API terminal client with response diffing",
		Long:    "Modern developer API client with latency benchmarks, syntax highlighting, assertions, and endpoint comparison",
	}

	httpCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom request header (key:value)")
	httpCmd.PersistentFlags().StringVarP(&data, "data", "d", "", "HTTP request body")
	httpCmd.PersistentFlags().StringVarP(&bearer, "bearer", "b", "", "Bearer authorization token")
	httpCmd.PersistentFlags().StringVarP(&timeoutStr, "timeout", "t", "15s", "Request timeout duration")
	httpCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Allow insecure TLS connections")
	httpCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show headers and timing breakdown")
	httpCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Save response payload to file")

	httpCmd.PersistentFlags().IntVar(&assertStatus, "assert-status", 0, "Assert expected HTTP status code (e.g. 200)")
	httpCmd.PersistentFlags().StringVar(&assertJSON, "assert-json", "", "Assert JSONPath field value (e.g. '.status=healthy')")
	httpCmd.PersistentFlags().StringVar(&assertMaxLatency, "assert-max-latency", "", "Assert response latency threshold (e.g. '300ms', '1s')")

	makeMethodCmd := func(method string) *cobra.Command {
		return &cobra.Command{
			Use:   strings.ToLower(method) + " <url>",
			Short: fmt.Sprintf("Perform HTTP %s request", method),
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				timeout, _ := time.ParseDuration(timeoutStr)
				client := httpclient.NewClient(timeout, insecure)

				hdrMap := parseHeaders(headers)
				opts := httpclient.RequestOptions{
					Method:      method,
					URL:         args[0],
					Headers:     hdrMap,
					Body:        []byte(data),
					BearerToken: bearer,
					Timeout:     timeout,
					InsecureTLS: insecure,
				}

				resp, err := client.Execute(opts)
				if err != nil {
					return fmt.Errorf("request failed: %w", err)
				}

				if outputFile != "" {
					if err := os.WriteFile(outputFile, resp.Body, 0644); err != nil {
						return fmt.Errorf("failed saving output to %s: %w", outputFile, err)
					}
					fmt.Printf("✓ Response saved to %s (%d bytes)\n", outputFile, len(resp.Body))
				}

				printResponse(resp, verbose)

				// Evaluate assertions if requested
				if assertStatus > 0 || assertJSON != "" || assertMaxLatency != "" {
					return evaluateAssertions(resp, assertStatus, assertJSON, assertMaxLatency)
				}
				return nil
			},
		}
	}

	requestCmd := &cobra.Command{
		Use:   "request <METHOD> <URL>",
		Short: "Execute custom HTTP request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			timeout, _ := time.ParseDuration(timeoutStr)
			client := httpclient.NewClient(timeout, insecure)

			opts := httpclient.RequestOptions{
				Method:      method,
				URL:         args[1],
				Headers:     parseHeaders(headers),
				Body:        []byte(data),
				BearerToken: bearer,
				Timeout:     timeout,
				InsecureTLS: insecure,
			}

			resp, err := client.Execute(opts)
			if err != nil {
				return err
			}

			if outputFile != "" {
				_ = os.WriteFile(outputFile, resp.Body, 0644)
			}

			printResponse(resp, verbose)

			if assertStatus > 0 || assertJSON != "" || assertMaxLatency != "" {
				return evaluateAssertions(resp, assertStatus, assertJSON, assertMaxLatency)
			}
			return nil
		},
	}

	diffCmd := &cobra.Command{
		Use:   "diff <url1> <url2>",
		Short: "Compare response structures and payloads of two HTTP endpoints",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			timeout, _ := time.ParseDuration(timeoutStr)
			client := httpclient.NewClient(timeout, insecure)

			opts := httpclient.RequestOptions{
				Method:      "GET",
				Headers:     parseHeaders(headers),
				BearerToken: bearer,
				Timeout:     timeout,
				InsecureTLS: insecure,
			}

			diff, err := client.CompareEndpoints(args[0], args[1], opts)
			if err != nil {
				return err
			}

			fmt.Print(httpclient.RenderDiffReport(diff))
			return nil
		},
	}

	httpCmd.AddCommand(
		makeMethodCmd("GET"),
		makeMethodCmd("POST"),
		makeMethodCmd("PUT"),
		makeMethodCmd("DELETE"),
		makeMethodCmd("PATCH"),
		makeMethodCmd("HEAD"),
		requestCmd,
		diffCmd,
	)

	return httpCmd
}

func evaluateAssertions(resp *httpclient.ResponseSummary, expectedStatus int, expectedJSON, maxLatency string) error {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	passStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	fmt.Println()
	fmt.Println(titleStyle.Render("🧪 API Health Assertions:"))
	fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────"))

	allPassed := true

	// 1. Status assertion
	if expectedStatus > 0 {
		if resp.StatusCode == expectedStatus {
			fmt.Printf("  %s Status Code: %d (expected %d)\n", passStyle.Render("✓"), resp.StatusCode, expectedStatus)
		} else {
			allPassed = false
			fmt.Printf("  %s Status Code: %d (expected %d)\n", failStyle.Render("✗"), resp.StatusCode, expectedStatus)
		}
	}

	// 2. Max latency assertion
	if maxLatency != "" {
		maxDur, err := time.ParseDuration(maxLatency)
		if err != nil {
			fmt.Printf("  %s Invalid --assert-max-latency duration: %v\n", failStyle.Render("✗"), err)
			allPassed = false
		} else {
			if resp.Timing.TotalTime <= maxDur {
				fmt.Printf("  %s Latency: %s <= %s\n", passStyle.Render("✓"), resp.Timing.TotalTime.Round(time.Millisecond), maxDur)
			} else {
				allPassed = false
				fmt.Printf("  %s Latency Exceeded: %s > %s\n", failStyle.Render("✗"), resp.Timing.TotalTime.Round(time.Millisecond), maxDur)
			}
		}
	}

	// 3. JSONPath assertion (.path=expectedVal)
	if expectedJSON != "" {
		parts := strings.SplitN(expectedJSON, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("  %s Invalid --assert-json format: %q (expected .path=value)\n", failStyle.Render("✗"), expectedJSON)
			allPassed = false
		} else {
			path := strings.TrimSpace(parts[0])
			expectedVal := strings.TrimSpace(parts[1])

			val, err := jsonview.QueryFromReader(bytes.NewReader(resp.Body), path)
			if err != nil {
				allPassed = false
				fmt.Printf("  %s JSONPath %s error: %v\n", failStyle.Render("✗"), path, err)
			} else {
				formattedVal, _ := jsonview.FormatQueryResult(val, false)
				cleanFormatted := strings.Trim(formattedVal, `"`)
				cleanExpected := strings.Trim(expectedVal, `"`)

				if cleanFormatted == cleanExpected {
					fmt.Printf("  %s JSONPath: %s == %q\n", passStyle.Render("✓"), path, cleanFormatted)
				} else {
					allPassed = false
					fmt.Printf("  %s JSONPath: %s == %q (expected %q)\n", failStyle.Render("✗"), path, cleanFormatted, cleanExpected)
				}
			}
		}
	}

	fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────"))
	if !allPassed {
		return fmt.Errorf("API assertions failed")
	}
	fmt.Println(dimStyle.Render("All health check assertions passed successfully."))
	return nil
}

func parseHeaders(hdrs []string) map[string]string {
	out := make(map[string]string)
	for _, h := range hdrs {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return out
}

func printResponse(r *httpclient.ResponseSummary, verbose bool) {
	statusStyle := lipgloss.NewStyle().Bold(true)
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#04B575"))
	} else if r.StatusCode >= 300 && r.StatusCode < 400 {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#00BFFF"))
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#FF5555"))
	}

	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	hdrKeyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))

	fmt.Printf("HTTP %s %s (%s)\n",
		statusStyle.Render(fmt.Sprintf("%d %s", r.StatusCode, r.StatusText)),
		metaStyle.Render(fmt.Sprintf("[%d bytes]", r.ContentLength)),
		metaStyle.Render(fmt.Sprintf("in %s", r.Timing.TotalTime.Round(time.Millisecond))),
	)

	if verbose {
		fmt.Println(metaStyle.Render("─────────────────── Headers ───────────────────"))
		for k, v := range r.Headers {
			fmt.Printf("%s: %s\n", hdrKeyStyle.Render(k), strings.Join(v, ", "))
		}

		fmt.Println(metaStyle.Render("─────────────────── Timing Waterfall ───────────"))
		fmt.Printf("  • DNS Lookup : %s\n", r.Timing.DNSDuration.Round(time.Microsecond))
		fmt.Printf("  • TCP Connect: %s\n", r.Timing.ConnDuration.Round(time.Microsecond))
		if r.Timing.TLSDuration > 0 {
			fmt.Printf("  • TLS Handshake: %s\n", r.Timing.TLSDuration.Round(time.Microsecond))
		}
		fmt.Printf("  • TTFB       : %s\n", r.Timing.TTFB.Round(time.Microsecond))
		fmt.Printf("  • Total Time : %s\n", r.Timing.TotalTime.Round(time.Microsecond))
		fmt.Println(metaStyle.Render("───────────────────────────────────────────────"))
	}

	if len(r.Body) > 0 {
		if r.IsJSON {
			var val interface{}
			if err := json.Unmarshal(r.Body, &val); err == nil {
				if pretty, err := json.MarshalIndent(val, "", "  "); err == nil {
					fmt.Println(string(pretty))
					return
				}
			}
		}
		fmt.Println(string(r.Body))
	}
}
