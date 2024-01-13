package resource

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const defaultTimeout time.Duration = 30 * time.Second
const schemeSecure string = "https"

var standardTransport *http.Transport = &http.Transport{
	DisableCompression: true,
	DisableKeepAlives:  true,
	IdleConnTimeout:    1 * time.Second,
	MaxIdleConns:       1,
}
var ErrNoHttpRequestSet error = errors.New("Request.httpreq not set, check request url")

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}

type Request struct {
	chResult chan *Result
	client   Requester
	httpreq  *http.Request
	timeout  time.Duration
	url      string
}

func (r *Request) Exec() *Result {
	if r.httpreq == nil {
		return &Result{Err: ErrNoHttpRequestSet}
	}

	requestStart := time.Now()
	resp, err := r.client.Do(r.httpreq)

	reqResult := &Result{
		Err:          err,
		RequestedUrl: r.httpreq.URL.String(),
		Response:     resp,
		Timing:       time.Since(requestStart),
	}
	reqResult.SetStatusFromResponse()

	return reqResult
}

func (r *Request) ExecAsync(ch chan *Result) {
	ch <- r.Exec()
}

func (r *Request) GetClient() Requester {
	return r.client
}

func (r *Request) SetRedirectsToPrint() {
	r.client.(*http.Client).CheckRedirect = func(req *http.Request, via []*http.Request) error {
		fmt.Printf(
			"%4sredirect! got %d, now requesting: %s\n",
			" ",
			req.Response.StatusCode, req.URL.String(),
		)

		return nil
	}
}

func (r *Request) SetupGet(u string) error {
	r.url = u
	formattedUrl, err := url.Parse(r.url)
	if err != nil {
		return err
	}

	if r.UsesHttpClient() && formattedUrl.Scheme != schemeSecure {
		r.client.(*http.Client).Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	req, err := http.NewRequest(http.MethodGet, formattedUrl.String(), nil)
	if err != nil {
		return err
	}

	r.httpreq = req

	return nil
}

func (r *Request) UnsetCheckRedirect() {
	r.client.(*http.Client).CheckRedirect = nil
}

func (r *Request) UsesHttpClient() bool {
	return reflect.TypeOf(r.client).String() == "*http.Client"
}

func NewRequest(u string) (*Request, error) {
	r := &Request{
		chResult: make(chan *Result, 1),
		client: &http.Client{
			Timeout:   defaultTimeout,
			Transport: standardTransport,
		},
	}

	if err := r.SetupGet(u); err != nil {
		return nil, err
	}

	return r, nil
}

func _NewRequest(options ...requestOption) (*Request, error) {
	r := &Request{
		chResult: make(chan *Result, 1),
		client: &http.Client{
			Timeout:   defaultTimeout,
			Transport: standardTransport,
		},
		timeout: defaultTimeout,
	}

	for _, o := range options {
		o.ApplyOption(r)
	}

	if r.UsesHttpClient() && r.timeout != defaultTimeout {
		r.client.(*http.Client).Timeout = r.timeout
	}

	if len(strings.TrimSpace(r.url)) > 0 {
		if err := r.SetupGet(r.url); err != nil {
			return nil, err
		}
	}

	return r, nil
}
