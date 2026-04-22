//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if err := waitForService(baseURL()+"/ping", readyDeadline()); err != nil {
		fmt.Fprintf(os.Stderr, "service not ready: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func readyDeadline() time.Time {
	d := 90 * time.Second
	if s := os.Getenv("INTEGRATION_READY_TIMEOUT"); s != "" {
		if dur, err := time.ParseDuration(s); err == nil && dur > 0 {
			d = dur
		}
	}
	return time.Now().Add(d)
}

func waitForService(url string, deadline time.Time) error {
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timed out waiting for %s", url)
}
