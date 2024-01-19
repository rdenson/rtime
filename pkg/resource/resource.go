package resource

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// processor function signature to handle making a request
// and returning our "wrapped" response
type RequestMaker func(c *http.Client, u string) *RequestResult

const schemeSecure string = "https"

// gets an http client to reuse in requests
func GetHttpClient(options ...httpClientOption) *http.Client {
	s := newHttpClientSettings()
	for _, o := range options {
		o.ApplyOption(s)
	}

	c := &http.Client{
		Timeout: s.timeout,
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  true,
			IdleConnTimeout:    1 * time.Second,
			MaxIdleConns:       1,
		},
	}

	return c
}

// does the actual request making
//
// Tries to vet the url and http request then builds out a
// "request response" based on what we got back.
func MakeRequest(c *http.Client, u string) *RequestResult {
	localClient := *c
	result := &RequestResult{
		initialUrl: u,
	}
	formattedUrl, err := url.Parse(u)
	if err != nil {
		result.err = err
		return result
	}

	if formattedUrl.Scheme != schemeSecure {
		localClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	req, err := http.NewRequest(http.MethodGet, formattedUrl.String(), nil)
	if err != nil {
		result.err = err
		return result
	}

	requestStart := time.Now()
	result.response, result.err = localClient.Do(req)
	result.timing = time.Since(requestStart)

	return result
}
