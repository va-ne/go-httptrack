# go-httptrack

`go-httptrack` is a small package that utilizes [`httptrace`](https://golang.org/pkg/net/http/httptrace/) to trace the events within HTTP client requests. By simply providing a `go-httptrack` context to the `http.Request` the tracing kicks in and we will get durations for DNS Lookout, TCP Connection, TLS Handshake, etc.
