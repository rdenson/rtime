package resource

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	parsedUrl, err := url.Parse(rtc.url)
	if err != nil {
		return nil, err
	}

	r := &Request{
		// Url:    parsedUrl,
		client: fhc,
	}

	req, err := http.NewRequest(http.MethodGet, parsedUrl.String(), nil)
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
			name:       "returns request error captured in result",
			url:        fixtureUrl,
			requestErr: errHttpClientDo,
			expects: &Result{
				Err:          errHttpClientDo,
				RequestedUrl: fixtureUrl,
			},
		},
		{
			name:         "returns response information captured in result",
			url:          fixtureUrl,
			responseCode: http.StatusOK,
			expects: &Result{
				Status:       http.StatusOK,
				RequestedUrl: fixtureUrl,
			},
		},
		// {
		// 	name:         "foo",
		// 	url:          fixtureUrl,
		// 	responseCode: http.StatusOK,
		// 	expects: &Result{
		// 		RequestStatus: http.StatusOK,
		// 		ResourceUrl:   fixtureUrl,
		// 	},
		// },
	}

	for _, scenario := range testCases {
		suite.Run(scenario.name, func() {
			r, err := scenario.getRequest()
			if err != nil {
				suite.Nil(err)
				return
			}

			res := r.Exec()

			// shim to help with scenario.expects an result equality comparison
			scenario.expects.(*Result).SetTiming(res.Timing)

			// vet Result
			suite.Equal(1, len(r.GetClient().(*httpClientMock).requestsReceived))
			suite.Equal(scenario.requestErr, res.Err)
			suite.Equal(scenario.expects, res)
		})
	}
}

func (suite *requestTestSuite) TestExecAsync() {
	tc := requestTestCase{
		url:          fixtureUrl,
		responseCode: http.StatusOK,
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
}
