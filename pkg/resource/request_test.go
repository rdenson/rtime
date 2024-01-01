package resource

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type requestTestCase struct {
	name         string
	url          string
	requestErr   error
	responseCode int
	expects      any
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

func (rtc *requestTestCase) getRequest() (*Request, error) {
	fhc := fakeHttpClient(
		"some body content",
		rtc.responseCode,
		http.Header{
			"Content-Type": []string{"text/html", "charset=utf-8"},
		},
		rtc.requestErr,
	)
	r := &Request{
		Url:     rtc.url,
		client:  fhc,
		httpreq: nil,
	}

	req, err := http.NewRequest("GET", r.Url, nil)
	if err != nil {
		return nil, err
	}

	r.httpreq = req

	return r, nil
}

// http.Client.Do() faking skaffold
//
// httpClientMock satisfies http.Client
// readCloserMock satisfies the body reader
func fakeHttpClient(responseContent string, responseStatusCode int, headers http.Header, errorToSimulate error) *httpClientMock {
	bytesBuffer := bytes.NewBufferString(responseContent)
	hcm := &httpClientMock{
		doMock: func(req *http.Request) (*http.Response, error) {
			if errorToSimulate != nil {
				return nil, errorToSimulate
			}

			return &http.Response{
				Body:          readCloserMock{bytesBuffer},
				Close:         true,
				ContentLength: int64(bytesBuffer.Len()),
				Header:        headers,
				Status:        "mocked_response",
				StatusCode:    responseStatusCode,
			}, nil
		},
		requestsReceived: make([]http.Request, 0),
	}

	return hcm
}

// testing fixure
var (
	fixtureUrl      string = "http://www.example.com"
	errHttpClientDo error  = fmt.Errorf("error in http.Client.do()")
)

func TestRequest(t *testing.T) {
	suite.Run(t, new(requestTestSuite))
}

func (suite *requestTestSuite) TestExec() {
	testCases := []requestTestCase{
		{
			name:       "returns request error captured in result",
			url:        fixtureUrl,
			requestErr: errHttpClientDo,
			expects: &Result{
				RequestErr:  errHttpClientDo,
				ResourceUrl: fixtureUrl,
			},
		},
		{
			name:         "returns response information captured in result",
			url:          fixtureUrl,
			responseCode: http.StatusOK,
			expects: &Result{
				RequestStatus: http.StatusOK,
				ResourceUrl:   fixtureUrl,
			},
		},
	}

	for _, scenario := range testCases {
		suite.Run(scenario.name, func() {
			r, err := scenario.getRequest()
			if err != nil {
				suite.Nil(err)
				return
			}

			_, res := r.Exec()

			// shim to help with scenario.expects an result equality comparison
			scenario.expects.(*Result).SetTiming(res.Timing)

			// vet Result
			suite.Equal(1, len(r.GetClient().(*httpClientMock).requestsReceived))
			suite.Equal(scenario.requestErr, res.RequestErr)
			suite.Equal(scenario.expects, res)
		})
	}
}

func (suite *requestTestSuite) TestExecAsync() {
	tc := requestTestCase{
		url:          fixtureUrl,
		responseCode: http.StatusOK,
	}

	r, err := tc.getRequest()
	if err != nil {
		suite.Nil(err)
		return
	}

	c := make(chan *Result, 1)
	w := new(sync.WaitGroup)

	go func(t *testing.T, ch chan *Result) {
		t.Log("listening for request results...")
		for {
			data, isOpen := <-ch
			if !isOpen {
				break
			}

			t.Logf("received: %+v", data)
		}
	}(suite.T(), c)

	w.Add(1)
	r.ExecAsync(tc.url, c, w)
	w.Wait()
	close(c)

	suite.T().Log("done waiting")
}
