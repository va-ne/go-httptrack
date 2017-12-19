package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

type Track struct {
	request               *http.Request
	isReused              bool
	isTLS                 bool
	dnsStart              time.Time
	dnsDone               time.Time
	DNSLookout            time.Duration
	connStart             time.Time
	connDone              time.Time
	Connect               time.Duration
	tlsHandshakeStart     time.Time
	tlsHandshakeDone      time.Time
	TLSHandshake          time.Duration
	serverProcessingStart time.Time
	serverProcessingDone  time.Time
	ServerProcessing      time.Duration
	Total                 time.Duration
}

func (t *Track) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("RoundTrip...")
	t.request = req
	return http.DefaultTransport.RoundTrip(req)
}

func (t *Track) getConnHandler(info httptrace.GotConnInfo) {
	fmt.Println("getConn...")

	t.isReused = info.Reused
}

func (t *Track) dnsStartHandler(info httptrace.DNSStartInfo) {
	fmt.Println("dnsStart...")

	t.dnsStart = time.Now()
}

func (t *Track) dnsDoneHandler(info httptrace.DNSDoneInfo) {
	fmt.Println("dnsDone...")

	t.dnsDone = time.Now()
	t.DNSLookout = t.dnsDone.Sub(t.dnsStart)
}

func (t *Track) connectStartHandler(_, _ string) {
	fmt.Println("connectStart...")

	t.connStart = time.Now()
}

func (t *Track) connectDoneHandler(network, addr string, err error) {
	fmt.Println("connectDone...")

	t.connDone = time.Now()
	t.Connect = t.connDone.Sub(t.connStart)
}

func (t *Track) tlsHandshakeStartHandler() {
	fmt.Println("tlsHandshakeStart...")

	t.tlsHandshakeStart = time.Now()
	t.isTLS = true
}

func (t *Track) tlsHandshakeDoneHandler(_ tls.ConnectionState, _ error) {
	fmt.Println("tlsHandshakeDone...")

	t.tlsHandshakeDone = time.Now()
	t.TLSHandshake = t.tlsHandshakeDone.Sub(t.tlsHandshakeStart)

}

func (t *Track) wroteRequestHandler(info httptrace.WroteRequestInfo) {
	fmt.Println("wroteRequest...")

	t.serverProcessingStart = time.Now()
}

func (t *Track) gotFirstResponseByteHandler() {
	fmt.Println("gotFirstResponseByte...")

	t.serverProcessingDone = time.Now()
	t.ServerProcessing = t.serverProcessingDone.Sub(t.serverProcessingStart)

	t.Total = t.serverProcessingDone.Sub(t.dnsStart)
}

// WithHTTPTrack - a wrapper of httptrace.WithClientTrace. It records the
// time of each httptrace hooks.
func WithHTTPTrack(ctx context.Context, t *Track) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart:             t.dnsStartHandler,
		DNSDone:              t.dnsDoneHandler,
		ConnectStart:         t.connectStartHandler,
		ConnectDone:          t.connectDoneHandler,
		TLSHandshakeStart:    t.tlsHandshakeStartHandler,
		TLSHandshakeDone:     t.tlsHandshakeDoneHandler,
		GotConn:              t.getConnHandler,
		WroteRequest:         t.wroteRequestHandler,
		GotFirstResponseByte: t.gotFirstResponseByteHandler,
	})
}

func (t Track) durations() map[string]time.Duration {
	return map[string]time.Duration{
		"DNSLookout":       t.DNSLookout,
		"Connect":          t.Connect,
		"TLSHandshake":     t.TLSHandshake,
		"ServerProcessing": t.ServerProcessing,
		"Total":            t.Total,
	}
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("Usage: go run main.go URL")
	}
	req, err := http.NewRequest("GET", args[1], nil)
	if err != nil {
		log.Fatal(err)
	}

	var result Track
	ctx := WithHTTPTrack(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()

	fmt.Printf("%+v\n", result.durations())
}
