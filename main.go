package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	Url                string
	Timeout            int
	Warning            float32
	Critical           float32
	OutputInMs         bool
	InsecureSkipVerify bool
	TlsTimeout         int
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-http-perf-go",
			Short:    "Alternate version of http-perf",
			Keyspace: "sensu.io/plugins/sensu-http-perf-go/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "url",
			Env:       "CHECK_URL",
			Argument:  "url",
			Shorthand: "u",
			Default:   "http://localhost:80/",
			Usage:     "URL to test (default http://localhost:80/)",
			Value:     &plugin.Url,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "timeout",
			Env:       "CHECK_TIMEOUT",
			Argument:  "timeout",
			Shorthand: "T",
			Default:   15,
			Usage:     "Request timeout in seconds",
			Value:     &plugin.Timeout,
		},
		&sensu.PluginConfigOption[float32]{
			Path:      "warning",
			Env:       "CHECK_WARNING",
			Argument:  "warning",
			Shorthand: "w",
			Default:   1,
			Usage:     "Warning threshold, in seconds",
			Value:     &plugin.Warning,
		},
		&sensu.PluginConfigOption[float32]{
			Path:      "critical",
			Env:       "CHECK_CRITICAL",
			Argument:  "critical",
			Shorthand: "c",
			Default:   2,
			Usage:     "Critical threshold, in seconds",
			Value:     &plugin.Critical,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "output-in-ms",
			Env:       "CHECK_OUTPUT_IN_MS",
			Argument:  "output-in-ms",
			Shorthand: "m",
			Default:   false,
			Usage:     "Provide output in milliseconds (default false, display in seconds)",
			Value:     &plugin.OutputInMs,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "insecure-skip-verify",
			Env:       "CHECK_INSECURE_SKIP_VERIFY",
			Argument:  "insecure-skip-verify",
			Shorthand: "i",
			Default:   false,
			Usage:     "Skip TLS certificate verification (not recommended!)",
			Value:     &plugin.InsecureSkipVerify,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "tls-timeout",
			Env:       "CHECK_TLS_TIMEOUT",
			Argument:  "tls-timeout",
			Shorthand: "z",
			Default:   1000,
			Usage:     "TLS handshake timeout in milliseconds",
			Value:     &plugin.TlsTimeout,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	if len(plugin.Url) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--url or CHECK_URL environment variable is required")
	}

	// ensure the warning and critical thresholds are valid, warnings must be lower than criticals
	if plugin.Warning > plugin.Critical {
		return sensu.CheckStateWarning, fmt.Errorf("warning threshold must be lower than critical threshold")
	}

	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	req, _ := http.NewRequest("GET", plugin.Url, nil)

	var (
		startTime, connectStart, connectDone, dnsStart, dnsDone, tlsHandshakeStart, tlsHandshakeDone, gotConn, firstResponseByte time.Time
	)

	// Define the HTTP trace.
	trace := &httptrace.ClientTrace{
		DNSStart:          func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:           func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectStart:      func(_, _ string) { connectStart = time.Now() },
		ConnectDone:       func(_, _ string, _ error) { connectDone = time.Now() },
		TLSHandshakeStart: func() { tlsHandshakeStart = time.Now() },
		TLSHandshakeDone:  func(_ tls.ConnectionState, _ error) { tlsHandshakeDone = time.Now() },
		GotConn:           func(_ httptrace.GotConnInfo) { gotConn = time.Now() },
		GotFirstResponseByte: func() {
			firstResponseByte = time.Now()
		},
	}

	// Associate the trace with the request context.
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second, // This is the TCP connection timeout
		}).DialContext,
		TLSHandshakeTimeout: time.Duration(plugin.TlsTimeout) * time.Millisecond,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: plugin.InsecureSkipVerify,
		},
	}

	client := &http.Client{
		Timeout:   time.Duration(plugin.Timeout) * time.Second, // This is the client timeout
		Transport: transport,
	}

	// Send the request and record the total time.
	startTime = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return sensu.CheckStateCritical, nil
	}

	defer resp.Body.Close()

	// Lets see if we completed the request with in the allowed time
	// Critical if we exceeded plugin.Critical and Warning if we exceeded plugin.Warning
	status := "OK"
	if time.Since(startTime) > time.Duration(plugin.Critical)*time.Second {
		status = "CRITICAL"
	} else if time.Since(startTime) > time.Duration(plugin.Warning)*time.Second {
		status = "WARNING"
	}

	// Output the results
	if !plugin.OutputInMs {
		fmt.Printf("%s %s: %.6fs | dns_duration=%.6f, tls_handshake_duration=%.6f, connect_duration=%.6f, first_byte_duration=%.6f, total_request_duration=%.6f\n",
			plugin.Name,
			status,
			time.Since(startTime).Seconds(),
			dnsDone.Sub(dnsStart).Seconds(),
			tlsHandshakeDone.Sub(tlsHandshakeStart).Seconds(),
			connectDone.Sub(connectStart).Seconds(),
			firstResponseByte.Sub(gotConn).Seconds(),
			time.Since(startTime).Seconds(),
		)
	} else {
		fmt.Printf("%s %s: %.6fms | dns_duration=%.2f, tls_handshake_duration=%.2f, connect_duration=%.2f, first_byte_duration=%.2f, total_request_duration=%.2f\n",
			plugin.Name,
			status,
			float64(time.Since(startTime))/float64(time.Millisecond),
			float64(dnsDone.Sub(dnsStart))/float64(time.Millisecond),
			float64(tlsHandshakeDone.Sub(tlsHandshakeStart))/float64(time.Millisecond),
			float64(connectDone.Sub(connectStart))/float64(time.Millisecond),
			float64(firstResponseByte.Sub(gotConn))/float64(time.Millisecond),
			float64(time.Since(startTime))/float64(time.Millisecond),
		)
	}
	if status == "CRITICAL" {
		return sensu.CheckStateCritical, nil
	} else if status == "WARNING" {
		return sensu.CheckStateWarning, nil
	}
	return sensu.CheckStateOK, nil
}
