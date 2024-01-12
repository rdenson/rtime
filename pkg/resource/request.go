package resource

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout time.Duration = 30 * time.Second
const schemeSecure string = "https"

var (
	RequestTimeout    time.Duration   = defaultTimeout
	insecureTransport *http.Transport = &http.Transport{
		DisableCompression: true,
		DisableKeepAlives:  true,
		IdleConnTimeout:    defaultTimeout,
		MaxIdleConns:       1,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	standardTransport *http.Transport = &http.Transport{
		DisableCompression: true,
		DisableKeepAlives:  true,
		IdleConnTimeout:    defaultTimeout,
		MaxIdleConns:       1,
	}
)

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}
type Request struct {
	client  Requester
	httpreq *http.Request
}

func (r *Request) Exec() (*http.Response, *Result) {
	requestStart := time.Now()
	response, doErr := r.client.Do(r.httpreq)

	reqResult := &Result{
		RequestErr:  doErr,
		ResourceUrl: r.httpreq.URL.String(),
		Timing:      time.Since(requestStart),
	}
	if doErr == nil {
		reqResult.RequestStatus = response.StatusCode
	}

	return response, reqResult
}

func (r *Request) ExecAsync(ch chan *Result) {
	requestStart := time.Now()
	resp, respErr := r.client.Do(r.httpreq)

	currentResult := &Result{
		RequestErr:  respErr,
		ResourceUrl: r.httpreq.URL.String(),
		Timing:      time.Since(requestStart),
	}
	if respErr == nil {
		currentResult.RequestStatus = resp.StatusCode
	}

	ch <- currentResult
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

func (r *Request) UnsetCheckRedirect() {
	r.client.(*http.Client).CheckRedirect = nil
}

func NewRequest(target string) (*Request, error) {
	r := &Request{
		client: &http.Client{
			Timeout:   RequestTimeout,
			Transport: standardTransport,
		},
	}

	formattedUrl, urlParseErr := url.Parse(target)
	if urlParseErr != nil {
		return nil, urlParseErr
	}

	if formattedUrl.Scheme != schemeSecure {
		r.client.(*http.Client).Transport = insecureTransport
	}

	req, newRequestErr := http.NewRequest(http.MethodGet, formattedUrl.String(), nil)
	if newRequestErr != nil {
		return nil, newRequestErr
	}

	r.httpreq = req

	return r, nil
}
