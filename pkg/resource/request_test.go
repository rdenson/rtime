package resource

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type requestTestCase struct {
	name    string
	url     string
	hclient *httpClientMock
	expects any
}

func (tc *requestTestCase) setHttpRequest() {
	var err error

	tc.expects.(*Request).httpreq, err = http.NewRequest(
		http.MethodGet,
		tc.expects.(*Request).url,
		nil,
	)
	if err != nil {
		panic(err)
	}
}

type requestTestSuite struct {
	suite.Suite
}

// tools faking an http client
type (
	httpClientMock struct {
		doMock           func(req *http.Request) (*http.Response, error)
		requestsReceived []http.Request
		requestWriter    sync.Mutex
		resp             *http.Response
	}
	readCloserMock struct {
		io.Reader
	}
)

func (hcm *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	hcm.requestWriter.Lock()
	hcm.requestsReceived = append(hcm.requestsReceived, *req)
	hcm.requestWriter.Unlock()

	return hcm.doMock(req)
}

func (rcm readCloserMock) Close() error {
	return nil
}

// http.Client.Do() faking skaffold
//
// httpClientMock satisfies http.Client
// readCloserMock satisfies the body reader
func fakeHttpClient(responseContent string, responseStatusCode int, headers http.Header, errorToSimulate error) *httpClientMock {
	bytesBuffer := bytes.NewBufferString(responseContent)
	r := &http.Response{
		Body:          readCloserMock{bytesBuffer},
		Close:         true,
		ContentLength: int64(bytesBuffer.Len()),
		Header:        headers,
		Status:        "mocked_response",
		StatusCode:    responseStatusCode,
	}
	hcm := &httpClientMock{
		doMock: func(req *http.Request) (*http.Response, error) {
			if errorToSimulate != nil {
				return nil, errorToSimulate
			}

			return r, nil
		},
		requestsReceived: make([]http.Request, 0),
		resp:             r,
	}

	return hcm
}

// testing fixure
var (
	errHttpClientDo           error        = fmt.Errorf("error in http.Client.do()")
	errUrlnvalid              error        = fmt.Errorf("net/url: invalid control character in URL")
	fixtureHtmlResponse       string       = "<!DOCTYPE html><html><body>some body content</body></html>"
	fixtureInsecureHttpClient *http.Client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  true,
			IdleConnTimeout:    1 * time.Second,
			MaxIdleConns:       1,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	fixtureStandardHttpClient *http.Client = &http.Client{
		Timeout:   defaultTimeout,
		Transport: standardTransport,
	}
	fixtureUrl             string = "http://example.com"
	fixtureUrlInsecure     string = "http://wwww.domain.com"
	fixtureUrlNoScheme     string = "example.com"
	fixtureUrlSecure       string = "https://example.com"
	fixtureUrlWithCrtlChar []byte = []byte{
		'h', 't', 't', 'p', 's', ':', '/', '/', 0x15,
	}
)

func TestRequest(t *testing.T) {
	suite.Run(t, new(requestTestSuite))
}

func (suite *requestTestSuite) TestFoo() {
	r, err := NewRequest(
		OptionRequestClient(
			fakeHttpClient(
				"some body content",
				http.StatusTeapot,
				http.Header{
					"Content-Type": []string{"text/html", "charset=utf-8"},
				},
				nil,
			),
		),
		OptionRequestUrl(fixtureUrl),
	)

	suite.Nil(err)
	suite.T().Logf("%+v", r)
	res := r.Exec()
	suite.T().Logf("result: %+v", res)

	suite.Equal(http.StatusTeapot, res.Status)
}

func (suite *requestTestSuite) TestBar() {
	r, err := NewRequest()
	suite.Nil(err)
	suite.T().Logf("%+v", r)

	res := r.Exec()
	suite.T().Logf("result: %+v", res)
	suite.Nil(res.Err)
}

func (suite *requestTestSuite) TestReal() {
	r, err := NewRequest(OptionRequestUrl("https://www.google.com"))
	suite.Nil(err)
	suite.T().Logf("%+v", r)

	res := r.Exec()
	suite.T().Logf("result: %+v", res)

	err = r.SetupGet("https://www.arstechnica.com")
	suite.Nil(err)

	res = r.Exec()
	suite.T().Logf("result: %+v", res)

	suite.True(true)
}

func (suite *requestTestSuite) TestExec() {
	testCases := []requestTestCase{
		{
			name: "returns error when exec called without http.Requst being set",
			hclient: fakeHttpClient(
				fixtureHtmlResponse,
				http.StatusOK,
				http.Header{
					"Content-Type": []string{"text/html", "charset=utf-8"},
				},
				nil,
			),
			expects: &Result{
				Err: ErrNoHttpRequestSet,
			},
		},
		{
			name: "returns request result",
			url:  fixtureUrl,
			hclient: fakeHttpClient(
				fixtureHtmlResponse,
				http.StatusOK,
				http.Header{
					"Content-Type": []string{"text/html", "charset=utf-8"},
				},
				nil,
			),
			expects: &Result{
				RequestedUrl: fixtureUrl,
				Status:       http.StatusOK,
			},
		},
		{
			name: "returns request error captured in result",
			url:  fixtureUrl,
			hclient: fakeHttpClient(
				// response data doesn't matter here
				// if r.client.Do(...) returns an error, the request did not execute
				"",
				0,
				nil,
				errHttpClientDo,
			),
			expects: &Result{
				Err:          errHttpClientDo,
				RequestedUrl: fixtureUrl,
			},
		},
	}

	for _, scenario := range testCases {
		suite.Run(scenario.name, func() {
			optionset := []requestOption{OptionRequestClient(scenario.hclient)}
			if len(scenario.url) > 0 {
				optionset = append(optionset, OptionRequestUrl(scenario.url))
			}

			r, err := NewRequest(optionset...)
			suite.Nil(err)
			res := r.Exec()
			// shims to help with scenario.expects equality comparison
			scenario.expects.(*Result).SetTiming(res.Timing)
			if res.Response != nil {
				scenario.expects.(*Result).Response = r.GetClient().(*httpClientMock).resp
			}

			suite.Equal(scenario.expects, res)
		})
	}
}

func (suite *requestTestSuite) TestSetupGet() {
	testCases := []requestTestCase{
		{
			name: "sets scheme for request if not set",
			url:  fixtureUrlNoScheme,
			expects: &Request{
				client:  fixtureStandardHttpClient,
				timeout: defaultTimeout,
				url:     fixtureUrlSecure,
			},
		},
		{
			name: "sets insecure transport for insecure url",
			url:  fixtureUrlInsecure,
			expects: &Request{
				client:  fixtureInsecureHttpClient,
				timeout: defaultTimeout,
				url:     fixtureUrlInsecure,
			},
		},
		{
			name:    "returns error if url cannot be parsed",
			url:     string(fixtureUrlWithCrtlChar),
			expects: errUrlnvalid,
		},
	}

	for _, scenario := range testCases {
		suite.Run(scenario.name, func() {
			r, err := NewRequest()
			suite.Nil(err)

			err = r.SetupGet(scenario.url)
			if err != nil {
				suite.Equal(scenario.expects, err.(*url.Error).Err)
			} else {
				suite.Nil(err)
				// shim to help with scenario.expects equality comparison
				scenario.setHttpRequest()

				suite.Equal(scenario.expects, r)
			}
		})
	}
}

/*func (suite *requestTestSuite) TestExecAsync() {
	tc := requestTestCase{
		url: fixtureUrl,
		// responseCode: http.StatusOK,
		expects: &Result{
			RequestedUrl: fixtureUrl,
			Status:       http.StatusOK,
		},
	}

	r, err := tc.getRequest()
	if err != nil {
		suite.Nil(err)
		return
	}

	c := make(chan *Result, 1)
	dc := make(chan bool, 1)
	w := new(sync.WaitGroup)

	go func(t *testing.T, ch chan *Result, wg *sync.WaitGroup) {
		t.Log("listening for request results...")
		for {
			data, isOpen := <-ch
			if !isOpen {
				break
			}

			wg.Done()
			t.Logf("received: %+v", data)
			tc.expects.(*Result).SetTiming(data.Timing)
			suite.Equal(tc.expects, data)
			dc <- true
		}
	}(suite.T(), c, w)

	w.Add(1)
	suite.T().Log("making asynchronous request")
	r.ExecAsync(c)
	w.Wait()
	close(c)

	<-dc
	suite.T().Log("done waiting")
}*/
