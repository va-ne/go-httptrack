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
	DNSStart              time.Time
	DNSDone               time.Time
	DNSLookout            time.Duration
	TCPStart              time.Time
	TCPDone               time.Time
	TCPConnection         time.Duration
	TLSHandshakeStart     time.Time
	TLSHandshakeDone      time.Time
	TLSHandshake          time.Duration
	ServerProcessingStart time.Time
	ServerProcessingDone  time.Time
	ServerProcessing      time.Duration
	Total                 time.Duration
}

func (t *Track) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("RoundTrip...")
	t.request = req
	return http.DefaultTransport.RoundTrip(req)
}

func (t *Track) getConn(info httptrace.GotConnInfo) {
	fmt.Println("getConn...")

	t.isReused = info.Reused
}

func (t *Track) dnsStart(info httptrace.DNSStartInfo) {
	fmt.Println("dnsStart...")

	t.DNSStart = time.Now()
}

func (t *Track) dnsDone(info httptrace.DNSDoneInfo) {
	fmt.Println("dnsDone...")

	t.DNSDone = time.Now()
	t.DNSLookout = t.DNSDone.Sub(t.DNSStart)
}

func (t *Track) connectStart(_, _ string) {
	fmt.Println("connectStart...")

	t.TCPStart = time.Now()
}

func (t *Track) connectDone(network, addr string, err error) {
	fmt.Println("connectDone...")

	t.TCPDone = time.Now()
	t.TCPConnection = t.TCPDone.Sub(t.TCPStart)
}

func (t *Track) tlsHandshakeStart() {
	fmt.Println("tlsHandshakeStart...")

	t.TLSHandshakeStart = time.Now()
	t.isTLS = true
}

func (t *Track) tlsHandshakeDone(_ tls.ConnectionState, _ error) {
	fmt.Println("tlsHandshakeDone...")

	t.TLSHandshakeDone = time.Now()
	t.TLSHandshake = t.TLSHandshakeDone.Sub(t.TLSHandshakeStart)

}

func (t *Track) wroteRequest(info httptrace.WroteRequestInfo) {
	fmt.Println("wroteRequest...")

	t.ServerProcessingStart = time.Now()
}

func (t *Track) gotFirstResponseByte() {
	fmt.Println("gotFirstResponseByte...")

	t.ServerProcessingDone = time.Now()
	t.ServerProcessing = t.ServerProcessingDone.Sub(t.ServerProcessingStart)

	// Check if total should be calc here
	t.Total = t.ServerProcessingDone.Sub(t.DNSStart)
}

func withClientTrace(ctx context.Context, t *Track) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart:             t.dnsStart,
		DNSDone:              t.dnsDone,
		ConnectStart:         t.connectStart,
		ConnectDone:          t.connectDone,
		TLSHandshakeStart:    t.tlsHandshakeStart,
		TLSHandshakeDone:     t.tlsHandshakeDone,
		GotConn:              t.getConn,
		WroteRequest:         t.wroteRequest,
		GotFirstResponseByte: t.gotFirstResponseByte,
	})
}

// WithHTTPTrack - a wrapper of httptrace.WithClientTrace. It records the
// time of each httptrace hooks.
func WithHTTPTrack(ctx context.Context, t *Track) context.Context {
	return withClientTrace(ctx, t)
}

func (t Track) Print() {
	fmt.Println("DNSLookout: ", t.DNSLookout)
	fmt.Println("TCPConnection: ", t.TCPConnection)
	fmt.Println("TLSHandshake: ", t.TLSHandshake)
	fmt.Println("ServerProcessing: ", t.ServerProcessing)
	fmt.Println("Total: ", t.Total)
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

	result.Print()
}
