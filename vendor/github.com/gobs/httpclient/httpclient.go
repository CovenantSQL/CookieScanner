package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gobs/pretty"
	"github.com/gobs/simplejson"
	//"net"
	//"github.com/jbenet/go-net-reuse"
)

var (
	DefaultClient = &http.Client{} // we use our own default client, so we can change the TLS configuration

	NoRedirect       = errors.New("No redirect")
	TooManyRedirects = errors.New("stopped after 10 redirects")
)

//
// Allow connections via HTTPS even if something is wrong with the certificate
// (self-signed or expired)
//
func AllowInsecure(insecure bool) {
	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		DefaultClient.Transport = tr
	} else {
		DefaultClient.Transport = nil
	}
}

//
// Set connection timeout
//
func SetTimeout(t time.Duration) {
	DefaultClient.Timeout = t
}

//
// HTTP error
//
type HttpError struct {
	Code       int
	Message    string
	RetryAfter int
	Body       []byte
	Header     http.Header
}

func (e HttpError) Error() string {
	if len(e.Body) > 0 {
		return fmt.Sprintf("%v %s", e.Message, e.Body)
	} else {
		return e.Message
	}
}

func (e HttpError) String() string {
	if len(e.Body) > 0 {
		return fmt.Sprintf("ERROR: %v %v %s", e.Code, e.Message, e.Body)
	} else {
		return fmt.Sprintf("ERROR: %v %v", e.Code, e.Message)
	}
}

//
// CloseResponse makes sure we close the response body
//
func CloseResponse(r *http.Response) {
	if r != nil && r.Body != nil {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}
}

//
// A wrapper for http.Response
//
type HttpResponse struct {
	http.Response
}

func (r *HttpResponse) ContentType() string {
	content_type := r.Header.Get("Content-Type")
	if len(content_type) == 0 {
		return content_type
	}

	return strings.TrimSpace(strings.Split(content_type, ";")[0])
}

//
// Close makes sure that all data from the body is read
// before closing the reader.
//
// If that is not the desider behaviour, just call HttpResponse.Body.Close()
//
func (r *HttpResponse) Close() {
	if r != nil {
		CloseResponse(&r.Response)
	}
}

//
// ResponseError checks the StatusCode and return an error if needed.
// The error is of type HttpError
//
func (r *HttpResponse) ResponseError() error {
	class := r.StatusCode / 100
	if class != 2 && class != 3 {
		rt := 0

		if h := r.Header.Get("Retry-After"); h != "" {
			rt, _ = strconv.Atoi(h)
		}

		var body [256]byte
		var blen int

		if r.Body != nil {
			blen, _ = r.Body.Read(body[:])
		}

		return HttpError{Code: r.StatusCode,
			Message:    "HTTP " + r.Status,
			RetryAfter: rt,
			Header:     r.Header,
			Body:       body[:blen],
		}
	}

	return nil
}

//
// CheckStatus returns err if not null or an HTTP status if the response was not "succesfull"
//
// usage:
//    resp, err := httpclient.CheckStatus(httpclient.SendRequest(params...))
//
func CheckStatus(r *HttpResponse, err error) (*HttpResponse, error) {
	if err != nil {
		return r, err
	}

	return r, r.ResponseError()
}

//
// Check if the input value is a "primitive" that can be safely stringified
//
func canStringify(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

//
// Given a base URL and a bag of parameteters returns the URL with the encoded parameters
//
func URLWithPathParams(base string, path string, params map[string]interface{}) (u *url.URL) {

	u, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}

	if len(path) > 0 {
		u, err = u.Parse(path)
		if err != nil {
			log.Fatal(err)
		}
	}

	q := u.Query()

	for k, v := range params {
		val := reflect.ValueOf(v)

		switch val.Kind() {
		case reflect.Slice:
			if val.IsNil() { // TODO: add an option to ignore empty values
				q.Set(k, "")
				continue
			}
			fallthrough

		case reflect.Array:
			for i := 0; i < val.Len(); i++ {
				av := val.Index(i)

				if canStringify(av) {
					q.Add(k, fmt.Sprintf("%v", av))
				}
			}

		default:
			if canStringify(val) {
				q.Set(k, fmt.Sprintf("%v", v))
			} else {
				log.Fatal("Invalid type ", val)
			}
		}
	}

	u.RawQuery = q.Encode()
	return u
}

func URLWithParams(base string, params map[string]interface{}) (u *url.URL) {
	return URLWithPathParams(base, "", params)
}

//
// http.Get with params
//
func Get(urlStr string, params map[string]interface{}) (*HttpResponse, error) {
	resp, err := DefaultClient.Get(URLWithParams(urlStr, params).String())
	if err == nil {
		return &HttpResponse{*resp}, nil
	} else {
		CloseResponse(resp)
		return nil, err
	}
}

//
// http.Post with params
//
func Post(urlStr string, params map[string]interface{}) (*HttpResponse, error) {
	resp, err := DefaultClient.PostForm(urlStr, URLWithParams(urlStr, params).Query())
	if err == nil {
		return &HttpResponse{*resp}, nil
	} else {
		CloseResponse(resp)
		return nil, err
	}
}

//
//  Read the body
//
func (resp *HttpResponse) Content() []byte {
	if resp == nil {
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err, " - read ", len(body), " bytes")
	}
	resp.Body.Close()
	return body
}

//
//  Try to parse the response body as JSON
//
func (resp *HttpResponse) Json() (json *simplejson.Json) {
	json, _ = simplejson.LoadBytes(resp.Content())
	return
}

//
// JsonDecode decodes the response body as JSON into specified structure
//
func (resp *HttpResponse) JsonDecode(out interface{}, strict bool) error {
	dec := json.NewDecoder(resp.Body)
	if strict {
		dec.DisallowUnknownFields()
	}
	defer resp.Body.Close()
	return dec.Decode(out)
}

////////////////////////////////////////////////////////////////////////

//
// http.Client with some defaults and stuff
//
type HttpClient struct {
	// the http.Client
	client *http.Client

	// the base URL for this client
	BaseURL *url.URL

	// overrides Host header
	Host string

	// the client UserAgent string
	UserAgent string

	// Common headers to be passed on each request
	Headers map[string]string

	// Cookies to be passed on each request
	Cookies []*http.Cookie

	// if FollowRedirects is false, a 30x response will be returned as is
	FollowRedirects bool

	// if HeadRedirects is true, the client will follow the redirect also for HEAD requests
	HeadRedirects bool

	// if Verbose, log request and response info
	Verbose bool

	// if Close, all requests will set Connection: close
	// (no keep-alive)
	Close bool
}

//
// Create a new HttpClient
//
func NewHttpClient(base string) (httpClient *HttpClient) {
	httpClient = new(HttpClient)
	httpClient.client = &http.Client{CheckRedirect: httpClient.checkRedirect}
	httpClient.Headers = make(map[string]string)
	httpClient.FollowRedirects = true

	if err := httpClient.SetBase(base); err != nil {
		log.Fatal(err)
	}

	return
}

// Set Base
//
//
func (self *HttpClient) SetBase(base string) error {
	u, err := url.Parse(base)
	if err != nil {
		return err
	}

	self.BaseURL = u
	return nil
}

//
// Set Transport
//
func (self *HttpClient) SetTransport(tr http.RoundTripper) {
	self.client.Transport = tr
}

//
// Get current Transport
//
func (self *HttpClient) GetTransport() http.RoundTripper {
	return self.client.Transport
}

//
// Set CookieJar
//
func (self *HttpClient) SetCookieJar(jar http.CookieJar) {
	self.client.Jar = jar
}

//
// Get current CookieJar
//
func (self *HttpClient) GetCookieJar() http.CookieJar {
	return self.client.Jar
}

//
// Allow connections via HTTPS even if something is wrong with the certificate
// (self-signed or expired)
//
func (self *HttpClient) AllowInsecure(insecure bool) {
	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			}}

		self.client.Transport = tr
	} else {
		self.client.Transport = nil
	}
}

//
// Set connection timeout
//
func (self *HttpClient) SetTimeout(t time.Duration) {
	self.client.Timeout = t
}

//
// Get connection timeout
//
func (self *HttpClient) GetTimeout() time.Duration {
	return self.client.Timeout
}

//
// Set LocalAddr in Dialer
// (this assumes you also want the SO_REUSEPORT/SO_REUSEADDR stuff)
//
/*
func (self *HttpClient) SetLocalAddr(addr string) {
	transport, ok := self.client.Transport.(*http.Transport)
	if transport == nil {
		if transport, ok = http.DefaultTransport.(*http.Transport); !ok {
			log.Println("SetLocalAddr for http.DefaultTransport != http.Transport")
			return
		}
	} else if !ok {
		log.Println("SetLocalAddr for client.Transport != http.Transport")
		return
	}
	if tcpaddr, err := net.ResolveTCPAddr("tcp", addr); err == nil {
		dialer := &reuse.Dialer{
			D: net.Dialer{
				Timeout:   30 * time.Second, // defaults from net/http DefaultTransport
				KeepAlive: 30 * time.Second, // defaults from net/http DefaultTransport
				LocalAddr: tcpaddr,
			}}
		transport.Dial = dialer.Dial
	} else {
		log.Println("Failed to resolve", addr, " to a TCP address")
	}
}
*/

//
// add default headers plus extra headers
//
func (self *HttpClient) addHeaders(req *http.Request, headers map[string]string) {

	if len(self.UserAgent) > 0 {
		req.Header.Set("User-Agent", self.UserAgent)
	}

	for k, v := range self.Headers {
		if _, add := headers[k]; !add {
			req.Header.Set(k, v)
		}
	}

	for _, c := range self.Cookies {
		req.AddCookie(c)
	}

	for k, v := range headers {
		if strings.ToLower(k) == "content-length" {
			if len, err := strconv.Atoi(v); err == nil && req.ContentLength <= 0 {
				req.ContentLength = int64(len)
			}
		} else {
			req.Header.Set(k, v)
		}
	}

}

//
// the callback for CheckRedirect, used to pass along the headers in case of redirection
//
func (self *HttpClient) checkRedirect(req *http.Request, via []*http.Request) error {
	if !self.FollowRedirects {
		// don't follow redirects if explicitly disabled
		return NoRedirect
	}

	if req.Method == "HEAD" && !self.HeadRedirects {
		// don't follow redirects on a HEAD request
		return NoRedirect
	}

	DebugLog(self.Verbose).Println("REDIRECT:", len(via), req.URL)
	if len(req.Cookies()) > 0 {
		DebugLog(self.Verbose).Println("COOKIES:", req.Cookies())
	}

	if len(via) >= 10 {
		return TooManyRedirects
	}

	if len(via) > 0 {
		last := via[len(via)-1]
		if len(last.Cookies()) > 0 {
			DebugLog(self.Verbose).Println("LAST COOKIES:", last.Cookies())
		}
	}

	// TODO: check for same host before adding headers
	self.addHeaders(req, nil)
	return nil
}

//
// Create a request object given the method, path, body and extra headers
//
func (self *HttpClient) Request(method string, urlpath string, body io.Reader, headers map[string]string) (req *http.Request) {
	if u, err := self.BaseURL.Parse(urlpath); err != nil {
		log.Fatal(err)
	} else {
		urlpath = u.String()
	}

	req, err := http.NewRequest(strings.ToUpper(method), urlpath, body)
	if err != nil {
		log.Fatal(err)
	}

	req.Close = self.Close
	req.Host = self.Host

	self.addHeaders(req, headers)

	return
}

////////////////////////////////////////////////////////////////////////////////////
//
// New style requests, with functional options

type RequestOption func(req *http.Request) (*http.Request, error)

// Set the request method
func Method(m string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		req.Method = strings.ToUpper(m)
		return req, nil
	}
}

var (
	HEAD   = Method("HEAD")
	GET    = Method("GET")
	POST   = Method("POST")
	PUT    = Method("PUT")
	DELETE = Method("DELETE")
)

// set the request URL
func URL(u *url.URL) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		req.URL = u
		return req, nil
	}
}

// set the request URL (passed as string)
func URLString(ustring string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		u, err := url.Parse(ustring)
		if err != nil {
			return nil, err
		}

		req.URL = u
		return req, nil
	}
}

// set the request path
func Path(path string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		if req.URL != nil {
			u, err := req.URL.Parse(path)
			if err != nil {
				return nil, err
			}

			req.URL = u
			return req, nil
		}

		u, err := url.Parse(path)
		if err != nil {
			return nil, err
		}

		req.URL = u
		return req, nil
	}
}

func (c *HttpClient) Path(path string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		u, err := c.BaseURL.Parse(path)
		if err != nil {
			return nil, err
		}

		req.URL = u
		return req, nil
	}
}

// set the request URL parameters
func Params(params map[string]interface{}) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		u := req.URL.String()
		req.URL = URLWithParams(u, params)
		return req, nil
	}
}

// set the request URL parameters
func StringParams(params map[string]string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
		return req, nil
	}
}

// set the request body as an io.Reader
func Body(r io.Reader) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		if r == nil {
			req.Body = http.NoBody
			req.ContentLength = 0
			return req, nil
		}

		if rc, ok := r.(io.ReadCloser); ok {
			req.Body = rc
		} else {
			req.Body = ioutil.NopCloser(r)
		}

		if v, ok := r.(interface{ Len() int }); ok {
			req.ContentLength = int64(v.Len())
		} else if v, ok := r.(interface{ Size() int64 }); ok {
			req.ContentLength = v.Size()
		}

		return req, nil
	}
}

// set the request body as a JSON object
func JsonBody(body interface{}) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		b, err := simplejson.DumpBytes(body)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
		req.ContentLength = int64(len(b))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		return req, nil
	}
}

// set the Accept header
func Accept(ct string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		req.Header.Set("Accept", ct)
		return req, nil
	}
}

// set the Content-Type header
func ContentType(ct string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		req.Header.Set("Content-Type", ct)
		return req, nil
	}
}

// set the Content-Length header
func ContentLength(l int64) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		if l >= 0 {
			req.ContentLength = l
		}
		return req, nil
	}
}

// set specified HTTP headers
func Header(headers map[string]string) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		for k, v := range headers {
			if strings.ToLower(k) == "content-length" {
				if len, err := strconv.Atoi(v); err == nil && req.ContentLength <= 0 {
					req.ContentLength = int64(len)
				}
			} else {
				req.Header.Set(k, v)
			}
		}

		return req, nil
	}
}

// set request context
func Context(ctx context.Context) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		return req.WithContext(ctx), nil
	}
}

// set request ClientTrace
func Trace(tracer *httptrace.ClientTrace) RequestOption {
	return func(req *http.Request) (*http.Request, error) {
		return req.WithContext(httptrace.WithClientTrace(req.Context(), tracer)), nil
	}
}

/* func Close(close bool) RequestOption {
	return func(req *http.Request) error {
		req.Close = close
		return nil
	}
} */

// Execute request
func (self *HttpClient) SendRequest(options ...RequestOption) (*HttpResponse, error) {
	req, err := http.NewRequest("GET", self.BaseURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Close = self.Close
	req.Host = self.Host

	self.addHeaders(req, nil)

	for _, opt := range options {
		if req, err = opt(req); err != nil {
			return nil, err
		}
	}

	return self.Do(req)
}

////////////////////////////////////////////////////////////////////////////////////
//
// Old style requests

//
// Execute request
//
func (self *HttpClient) Do(req *http.Request) (*HttpResponse, error) {
	DebugLog(self.Verbose).Println("REQUEST:", req.Method, req.URL, pretty.PrettyFormat(req.Header))

	resp, err := self.client.Do(req)
	if urlerr, ok := err.(*url.Error); ok && urlerr.Err == NoRedirect {
		err = nil // redirect on HEAD is not an error
	}
	if err == nil {
		DebugLog(self.Verbose).Println("RESPONSE:", resp.Status, pretty.PrettyFormat(resp.Header))
		return &HttpResponse{*resp}, nil
	} else {
		DebugLog(self.Verbose).Println("ERROR:", err,
			"REQUEST:", req.Method, req.URL,
			pretty.PrettyFormat(req.Header))
		CloseResponse(resp)
		return nil, err
	}
}

//
// Execute a DELETE request
//
func (self *HttpClient) Delete(path string, headers map[string]string) (*HttpResponse, error) {
	req := self.Request("DELETE", path, nil, headers)
	return self.Do(req)
}

//
// Execute a HEAD request
//
func (self *HttpClient) Head(path string, params map[string]interface{}, headers map[string]string) (*HttpResponse, error) {
	req := self.Request("HEAD", URLWithParams(path, params).String(), nil, headers)
	return self.Do(req)
}

//
// Execute a GET request
//
func (self *HttpClient) Get(path string, params map[string]interface{}, headers map[string]string) (*HttpResponse, error) {
	req := self.Request("GET", URLWithParams(path, params).String(), nil, headers)
	return self.Do(req)
}

//
// Execute a POST request
//
func (self *HttpClient) Post(path string, content io.Reader, headers map[string]string) (*HttpResponse, error) {
	req := self.Request("POST", path, content, headers)
	return self.Do(req)
}

func (self *HttpClient) PostForm(path string, data url.Values, headers map[string]string) (*HttpResponse, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	req := self.Request("POST", path, strings.NewReader(data.Encode()), headers)
	return self.Do(req)
}

//
// Execute a PUT request
//
func (self *HttpClient) Put(path string, content io.Reader, headers map[string]string) (*HttpResponse, error) {
	req := self.Request("PUT", path, content, headers)
	return self.Do(req)
}

//
// Upload a file via form
//
func (self *HttpClient) UploadFile(method, path, fileParam, filePath string, payload []byte, params map[string]string, headers map[string]string) (*HttpResponse, error) {
	var reader io.Reader

	if payload == nil {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		reader = file
	} else {
		reader = bytes.NewReader(payload)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileParam, filepath.Base(filePath))
	if err == nil {
		_, err = io.Copy(part, reader)
	}
	if err == nil {
		for key, val := range params {
			writer.WriteField(key, val)
		}
		err = writer.Close()
	}
	if err != nil {
		return nil, err
	}

	if headers == nil {
		headers = map[string]string{}
	}

	headers["Content-Type"] = writer.FormDataContentType()
	headers["Content-Length"] = strconv.Itoa(body.Len())
	req := self.Request(method, path, body, headers)

	return self.Do(req)
}
