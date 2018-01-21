package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

const (
	TestHTTP  = "http://example.com"
	TestHTTPS = "https://example.com"
)

func TestHTTPTrackHTTPS(t *testing.T) {
	var result Track

	req, err := http.NewRequest("GET", TestHTTPS, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := WithHTTPTrack(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	if !result.isTLS {
		t.Fatal("isTLS should be true")
	}

	for k, d := range result.durations() {
		if d <= 0*time.Millisecond {
			t.Fatalf("%s should be > 0", k)
		}
	}
}

func TestHTTPTrackHTTP(t *testing.T) {
	var result Track

	req, err := http.NewRequest("GET", TestHTTP, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := WithHTTPTrack(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	if result.isTLS {
		t.Fatal("isTLS should be false")
	}

	if got, want := result.TLSHandshake, 0*time.Millisecond; got != want {
		t.Fatalf("TLSHandshake time = %d, want %d", got, want)
	}

	// We are expecting 0 for TLSHandshake
	// remove it from the durations for the next check
	durations := result.durations()
	delete(durations, "TLSHandshake")

	for k, d := range durations {
		if d <= 0*time.Millisecond {
			t.Fatalf("%s should be > 0", k)
		}
	}
}

func TestHTTPTrackKeepAlive(t *testing.T) {
	req1, err := http.NewRequest("GET", TestHTTPS, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := http.DefaultClient
	res1, err := client.Do(req1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := io.Copy(ioutil.Discard, res1.Body); err != nil {
		t.Fatal(err)
	}
	res1.Body.Close()

	// Make second request
	// connection should be re-used
	var result Track
	req2, err := http.NewRequest("GET", TestHTTPS, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := WithHTTPTrack(req2.Context(), &result)
	req2 = req2.WithContext(ctx)

	client = http.DefaultClient
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal("FAIL: request - ", err)
	}

	if _, err := io.Copy(ioutil.Discard, res2.Body); err != nil {
		t.Fatal("FAIL: copy body - ")
	}
	res2.Body.Close()

	// DNSLookup, Connect, TLSHandshake should be 0
	durations := []time.Duration{
		result.DNSLookout,
		result.Connect,
		result.TLSHandshake,
	}

	for k, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Fatalf("#%d=%d expected to be eq %d", k, got, want)
		}
	}
}
