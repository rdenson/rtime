package resource

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const defaultTimeout time.Duration = 30 * time.Second

var (
	RequestTimeout    time.Duration   = defaultTimeout
	insecureTransport *http.Transport = &http.Transport{
		DisableCompression: true,
		IdleConnTimeout:    defaultTimeout,
		MaxIdleConns:       10,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	standardTransport *http.Transport = &http.Transport{
		DisableCompression: true,
		IdleConnTimeout:    defaultTimeout,
		MaxIdleConns:       10,
	}
)

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}
type Request struct {
	Url     string
	client  Requester
	httpreq *http.Request
}

func (r *Request) Exec() (*http.Response, *Result) {
	requestStart := time.Now()
	response, doErr := r.client.Do(r.httpreq)

	reqResult := &Result{
		RequestErr:  doErr,
		ResourceUrl: r.Url,
		Timing:      time.Since(requestStart),
	}
	if doErr == nil {
		reqResult.RequestStatus = response.StatusCode
	}

	return response, reqResult
}

func (r *Request) ExecAsync(target string, ch chan *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	req, newRequestErr := http.NewRequest("GET", target, nil)
	if newRequestErr != nil {
		// short circuit; incorrectly formated http.Request{}
		ch <- &Result{RequestErr: newRequestErr}
		return
	}

	req.Close = true
	requestStart := time.Now()
	resp, respErr := r.client.Do(req)
	currentResult := &Result{
		RequestErr:  respErr,
		ResourceUrl: target,
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

func NewRequest(target string, isSecure bool) (*Request, error) {
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

	formattedUrl.Scheme = "https"
	if !isSecure {
		r.client.(*http.Client).Transport = insecureTransport
		formattedUrl.Scheme = "http"
	}

	r.Url = formattedUrl.String()
	req, newRequestErr := http.NewRequest("GET", r.Url, nil)
	if newRequestErr != nil {
		return nil, newRequestErr
	}

	req.Close = true
	r.httpreq = req

	return r, nil
}
