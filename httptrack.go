package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
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
	t.request = req
	return http.DefaultTransport.RoundTrip(req)
}

func (t *Track) getConnHandler(info httptrace.GotConnInfo) {
	t.isReused = info.Reused
}

func (t *Track) dnsStartHandler(info httptrace.DNSStartInfo) {
	t.dnsStart = time.Now()
}

func (t *Track) dnsDoneHandler(info httptrace.DNSDoneInfo) {
	t.dnsDone = time.Now()
	t.DNSLookout = t.dnsDone.Sub(t.dnsStart)
}

func (t *Track) connectStartHandler(_, _ string) {
	t.connStart = time.Now()
}

func (t *Track) connectDoneHandler(network, addr string, err error) {
	t.connDone = time.Now()
	t.Connect = t.connDone.Sub(t.connStart)
}

func (t *Track) tlsHandshakeStartHandler() {
	t.tlsHandshakeStart = time.Now()
	t.isTLS = true
}

func (t *Track) tlsHandshakeDoneHandler(_ tls.ConnectionState, _ error) {
	t.tlsHandshakeDone = time.Now()
	t.TLSHandshake = t.tlsHandshakeDone.Sub(t.tlsHandshakeStart)
}

func (t *Track) wroteRequestHandler(info httptrace.WroteRequestInfo) {
	t.serverProcessingStart = time.Now()
}

func (t *Track) gotFirstResponseByteHandler() {
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
