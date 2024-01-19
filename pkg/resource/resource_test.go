package resource

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type resourceTestSuite struct {
	suite.Suite
}

type requestResultTestCase struct {
	name               string
	url                string
	useTestServer      bool
	responseBody       string
	responseHeaders    map[string]string
	responseStatusCode int
	expects            any

	_svr     *httptest.Server
	_hclient *http.Client
}

func (tc *requestResultTestCase) setup() {
	tc._hclient = GetHttpClient()
	if !tc.useTestServer {
		return
	}

	if tc.responseStatusCode == 0 {
		tc.responseStatusCode = http.StatusOK
	}

	tc._svr = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range tc.responseHeaders {
			w.Header().Add(k, v)
		}

		w.WriteHeader(tc.responseStatusCode)
		if len(tc.responseBody) > 0 {
			w.Write([]byte(tc.responseBody))
		}
	}))
	tc.url = tc._svr.URL
}

func (tc *requestResultTestCase) replacePlaceholdersInExpectedSummary(r *RequestResult) {
	tc.expects = strings.Replace(tc.expects.(string), testingPlaceholderUrl, tc.url, 1)
	tc.expects = strings.Replace(
		tc.expects.(string),
		testingPlaceholderTiming,
		fmt.Sprintf("%dms", r.GetTiming().Milliseconds()),
		1,
	)
}

const testingPlaceholderUrl string = "URL"
const testingPlaceholderTiming string = "0ms"

var (
	fixtureBadUrl             string = "/thing"
	fixtureContentTypeJson    string = "application/json"
	fixtureJsResource         string = "http://cdn.example.net/scripts/a.js"
	fixtureRelativeResource   string = "/css/default.css"
	fixtureTimingMilliseconds int64  = 0
	fixtureHtmlResponse       string = fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
		<head>
			<script type="text/javascript" src="%s">
			</script>
			<link rel="stylesheet\" href="%s">
		</head>
		<body>some body content</body>
	</html>
	`, fixtureJsResource, fixtureRelativeResource)
	fixtureJsonResponse string = `
	{
		"key0": "value0",
		"key1": "value1"
	}
	`
)

func TestResource(t *testing.T) {
	suite.Run(t, new(resourceTestSuite))
}

func (suite *resourceTestSuite) TestMakeRequest() {
	testCases := []*requestResultTestCase{
		{
			name: "returns url.Error for a bad url",
			url:  fixtureBadUrl,
			expects: &url.Error{
				Op:  "Get",
				URL: fixtureBadUrl,
				Err: errors.New("unsupported protocol scheme \"\""),
			},
		},
		{
			name:         "returns expected summary",
			responseBody: fixtureJsonResponse,
			responseHeaders: map[string]string{
				"Content-Type": fmt.Sprintf("%s; charset=utf-8", fixtureContentTypeJson),
			},
			responseStatusCode: http.StatusBadRequest,
			useTestServer:      true,
			expects: fmt.Sprintf(
				summaryFormat,
				testingPlaceholderUrl,
				fmt.Sprintf(formatContentType, fixtureContentTypeJson),
				http.StatusBadRequest,
				fixtureTimingMilliseconds,
			),
		},
		{
			name:               "returns expected summary without content type",
			responseStatusCode: http.StatusForbidden,
			useTestServer:      true,
			expects: fmt.Sprintf(
				summaryFormat,
				testingPlaceholderUrl,
				formatNoContentType,
				http.StatusForbidden,
				fixtureTimingMilliseconds,
			),
		},
	}

	for _, scenario := range testCases {
		suite.Run(scenario.name, func() {
			scenario.setup()

			res := MakeRequest(scenario._hclient, scenario.url)
			if res.HasError() {
				suite.Equal(scenario.expects, res.GetError())
			} else {
				scenario.replacePlaceholdersInExpectedSummary(res)
				suite.Equal(scenario.expects, res.Summarize())
			}

			if scenario.useTestServer {
				scenario._svr.Close()
			}
		})
	}
}
