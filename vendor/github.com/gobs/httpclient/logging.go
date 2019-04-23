package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"os"
	"strconv"
	"time"
)

// A transport that prints request and response

type LoggingTransport struct {
	t            *http.Transport
	requestBody  bool
	responseBody bool
	timing       bool
}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	dreq, _ := httputil.DumpRequest(req, lt.requestBody)
	//fmt.Println("REQUEST:", strconv.Quote(string(dreq)))
	fmt.Println("URL:", req.URL)
	fmt.Println("REQUEST:", string(dreq))

	for _, t := range req.TransferEncoding {
		fmt.Println("Transfer-Encoding:", t)
	}

	fmt.Println("")

	var startTime time.Time
	var elapsed time.Duration

	if lt.timing {
		startTime = time.Now()
	}

	resp, err = lt.t.RoundTrip(req)

	if lt.timing {
		elapsed = time.Since(startTime)
	}

	if err != nil {
		if lt.requestBody {
			// don't print the body twice
			dreq, _ = httputil.DumpRequest(req, false)
		}
		fmt.Println("ERROR:", err, "REQUEST:", strconv.Quote(string(dreq)))
	}
	if resp != nil {
		dresp, _ := httputil.DumpResponse(resp, lt.responseBody)
		fmt.Println("RESPONSE:", string(dresp))

		for _, t := range resp.Request.TransferEncoding {
			fmt.Println("RQ Transfer-Encoding:", t)
		}
	}

	if elapsed > 0 {
		fmt.Println("ELAPSED TIME:", elapsed.Round(time.Millisecond))
	}

	fmt.Println("")
	return
}

func (lt *LoggingTransport) CancelRequest(req *http.Request) {
	dreq, _ := httputil.DumpRequest(req, false)
	fmt.Println("CANCEL REQUEST:", strconv.Quote(string(dreq)))
	lt.t.CancelRequest(req)
}

// Enable logging requests/response headers
//
// if requestBody == true, also log request body
// if responseBody == true, also log response body
// if timing == true, also log elapsed time
func StartLogging(requestBody, responseBody, timing bool) {
	http.DefaultTransport = &LoggingTransport{&http.Transport{}, requestBody, responseBody, timing}
}

// Disable logging requests/responses
func StopLogging() {
	http.DefaultTransport = &http.Transport{}
}

// Wrap input transport into a LoggingTransport
func LoggedTransport(t *http.Transport, requestBody, responseBody, timing bool) http.RoundTripper {
	return &LoggingTransport{t, requestBody, responseBody, timing}
}

// A Reader that "logs" progress

type ProgressReader struct {
	r         io.Reader
	c         [1]byte
	threshold int
	curr      int
}

func NewProgressReader(r io.Reader, c byte, threshold int) *ProgressReader {
	if c == 0 {
		c = '.'
	}
	if threshold <= 0 {
		threshold = 10240
	}
	p := &ProgressReader{r: r, c: [1]byte{c}, threshold: threshold, curr: 0}
	return p
}

func (p *ProgressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)

	p.curr += n

	if err == io.EOF {
		os.Stdout.Write([]byte{'\n'})
		os.Stdout.Sync()
	} else if p.curr >= p.threshold {
		p.curr -= p.threshold

		os.Stdout.Write(p.c[:])
		os.Stdout.Sync()
	}

	return n, err
}

func (p *ProgressReader) Close() error {
	if rc, ok := p.r.(io.ReadCloser); ok {
		return rc.Close()
	} else {
		return nil
	}
}

// A Logger that can be disabled

type DebugLog bool

func (d DebugLog) Println(args ...interface{}) {
	if d {
		log.Println(args...)
	}
}

func (d DebugLog) Printf(fmt string, args ...interface{}) {
	if d {
		log.Printf(fmt, args...)
	}
}

// A ClientTrace implementation that collects request time

type RequestTrace struct {
	DNS          time.Duration
	Connect      time.Duration
	Connected    bool
	TLSHandshake time.Duration
	Request      time.Duration
	Wait         time.Duration
	Response     time.Duration

	Local  string
	Remote string

	startTime time.Time
}

func (r *RequestTrace) Reset() {
	r.DNS = 0
	r.Connect = 0
	r.Connected = false
	r.TLSHandshake = 0
	r.Request = 0
	r.Wait = 0
	r.Response = 0
}

func (r *RequestTrace) NewClientTrace(trace bool) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			r.Connected = info.WasIdle
			r.Local = info.Conn.LocalAddr().String()
			r.Remote = info.Conn.RemoteAddr().String()

			if trace {
				log.Println("GotConn", r.Remote, "reused:", info.Reused, "wasIdle:", info.WasIdle)
			}
		},

		ConnectStart: func(network, addr string) {
			r.startTime = time.Now()

			if trace {
				log.Println("ConnectStart", network, addr)
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.Connect = time.Since(r.startTime)
			r.startTime = time.Time{}

			if trace {
				log.Println("ConnectDone", network, addr, err, r.Connect)
			}
		},

		DNSStart: func(info httptrace.DNSStartInfo) {
			r.startTime = time.Now()

			if trace {
				log.Println("DNSStart", info.Host)
			}
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			r.DNS = time.Since(r.startTime)
			r.startTime = time.Time{}

			if trace {
				log.Println("DNSDone", info.Addrs, r.DNS)
			}
		},

		TLSHandshakeStart: func() {
			r.startTime = time.Now()

			if trace {
				log.Println("TLSHandshakeStart")
			}
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			r.TLSHandshake = time.Since(r.startTime)
			r.startTime = time.Time{}

			if trace {
				log.Println("TLSHandshakeDone", err, r.TLSHandshake)
			}
		},

		WroteHeaderField: func(string, []string) {
			if r.startTime.IsZero() {
				r.startTime = time.Now()

				if trace {
					log.Println("WroteHeader")
				}
			}
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.Request = time.Since(r.startTime)
			r.startTime = time.Now()

			if trace {
				log.Println("WroteRequest", r.Request)
			}
		},

		GotFirstResponseByte: func() {
			r.Wait = time.Since(r.startTime)
			r.startTime = time.Now()

			if trace {
				log.Println("GotFirstResponseByte", r.Wait)
			}
		},
	}
}

// Call this after the response has been received to set the Response duration
// (there is no callback for this)

func (r *RequestTrace) Done() {
	r.Response = time.Since(r.startTime)
	r.startTime = time.Now()
}

func (r *RequestTrace) String() string {
	return fmt.Sprintf("{DNS:%v, Connect:%v, Connected:%v, TLSHandshake:%v, Request:%v, Wait:%v, Response:%v}",
		r.DNS.Truncate(100*time.Microsecond),
		r.Connect.Truncate(100*time.Microsecond),
		r.Connected,
		r.TLSHandshake.Truncate(100*time.Microsecond),
		r.Request.Truncate(100*time.Microsecond),
		r.Wait.Truncate(100*time.Microsecond),
		r.Response.Truncate(100*time.Microsecond))
}
