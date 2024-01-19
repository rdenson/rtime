package main

import (
	"fmt"

	"github.com/rdenson/rtime/pkg/resource"
	"github.com/spf13/cobra"
)

/*
	var pageCmd = &cobra.Command{
		Use:   "page [url]",
		Short: "request a page, just as you would in your browser",
		Long: `Attempts to request the specified URL and resolve any associated
	  resources. eg. css, images, scripts, etc. Requests for additional
	  resources are made concurrently.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, newRequestErr := resource.NewRequest(resource.OptionRequestUrl(args[0]))
			if newRequestErr != nil {
				return newRequestErr
			}

			req.SetRedirectsToPrint()
			reqResult := req.Exec()
			reqResult.PrettyPrint()
			if reqResult.Err != nil {
				return reqResult.Err
			}

			fmt.Printf("status: %d\n", reqResult.Status)

			resourcesToResolve := getResourcesFromResponseBody(reqResult.Response.Body)
			fmt.Println("resolving resources...")
			timings := make([]*resource.Result, 0)
			timingCh := make(chan *resource.Result, 1)
			resourceWg := new(sync.WaitGroup)
			req.UnsetCheckRedirect()
			go func(wg *sync.WaitGroup) {
				for {
					reqResult, isOpen := <-timingCh
					if !isOpen {
						break
					}

					if reqResult.Err == nil {
						timings = append(timings, reqResult)
					}

					wg.Done()
				}
			}(resourceWg)
			initialClient := req.GetClient()
			for _, resourcePath := range resourcesToResolve {
				resourceWg.Add(1)
				// fmt.Printf(">>> requesting %s\n", resourcePath)
				// go req.ExecAsync(
				// 	timingCh,
				// )
				go func(u string, ch chan *resource.Result) {
					r, err := resource.NewRequest(
						resource.OptionRequestClient(initialClient),
						resource.OptionRequestUrl(u),
					)
					if err != nil {
						fmt.Printf("error requesting %s: %s\n", u, err.Error())
					}

					ch <- r.Exec()
				}(resourcePath, timingCh)
			}

			resourceWg.Wait()
			fmt.Printf(
				"%4sfinished requesting %d resource(s)\n",
				" ",
				len(timings),
			)
			close(timingCh)
			var largestResourceRequestTime time.Duration
			for _, t := range timings {
				if t.Timing > largestResourceRequestTime {
					largestResourceRequestTime = t.Timing
				}
			}

			fmt.Printf("%4slongest associated resource request time: %s\n", " ", largestResourceRequestTime)
			fmt.Printf("total request estimated at %s\n", reqResult.Timing+largestResourceRequestTime)

			if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
				showResponseHeaders(reqResult.Response)
			}

			if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
				showResponseTlsInfo(reqResult.Response)
			}

			if showResourceRequests, _ := cmd.Flags().GetBool("show-resources-requested"); showResourceRequests {
				fmt.Println()
				fmt.Println("resources parsed from initial request body:")
				for _, t := range timings {
					// fmt.Printf("%5s %s %5d - %s\n", " ", t.Timing, t.RequestStatus, t.ResourceUrl)
					t.PrettyPrint()
				}

				fmt.Println()
			}

			return nil
		},
	}
*/
var pageCmd = &cobra.Command{
	Use:   "page [url]",
	Short: "request a page, just as you would in your browser",
	Long: `Attempts to request the specified URL and resolve any associated
  resources. eg. css, images, scripts, etc. Requests for additional
  resources are made concurrently.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		showPageResources, _ := cmd.Flags().GetBool("show-resources-requested")
		hc := resource.GetHttpClient()
		res := resource.MakeRequest(hc, args[0])
		if res.HasError() {
			return res.GetError()
		}

		fmt.Printf(
			"%s responded in with %d in %.3fs\n",
			res.GetResponseUrl(),
			res.GetStatus(),
			res.GetTiming().Seconds(),
		)
		if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
			showResponseHeaders(res.GetHeaders())
		}

		fmt.Print("trying to resolve page resources... ")
		resourcesToResolve := res.GetResourcesInResponse()
		fmt.Printf("resources found: %d\n", len(resourcesToResolve))

		asyncResults := resource.NewRequestResultset(resource.MakeRequest)
		go asyncResults.ReceiveResults()
		for _, resourcePath := range resourcesToResolve {
			asyncResults.Exec(hc, resourcePath)
		}

		asyncResults.Wait()

		if showPageResources {
			errs := asyncResults.GetErroredResults()
			rok := asyncResults.GetSuccessfulResults()
			fmt.Println("resource requests")
			fmt.Printf("%*serrors (%d)\n", 2, " ", len(errs))
			for _, e := range errs {
				fmt.Printf("%*s%s\n", 2, " ", e.Summarize())
			}

			fmt.Printf("%*sresponses (%d)\n", 2, " ", len(rok))
			for _, r := range rok {
				fmt.Printf("%*s%s\n", 2, " ", r.Summarize())
			}
		}

		longestRequest := asyncResults.GetLongestRequest()
		fmt.Printf("longest request: %s\n", longestRequest.Summarize())
		fmt.Printf("total request estimated at %s\n", res.GetTiming()+longestRequest.GetTiming())
		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(res.GetTlsState())
		}

		return nil
	},
}

/*const schemeSecure string = "https"

type requestMaker func(c *http.Client, u string) *requestResult
type requestResultset struct {
	ch       chan *requestResult
	executor requestMaker
	results  []*requestResult
	wg       *sync.WaitGroup
}

func (r *requestResultset) exec(client *http.Client, u string) {
	r.wg.Add(1)
	go func() {
		r.ch <- r.executor(client, u)
	}()
}

func (r *requestResultset) getLongestRequest() *requestResult {
	var largestResourceRequestTime time.Duration
	var longestRequest *requestResult

	for _, res := range r.results {
		if res.timing > largestResourceRequestTime {
			largestResourceRequestTime = res.timing
			longestRequest = res
		}
	}

	return longestRequest
}

func (r *requestResultset) receiveResults(showSummary bool) {
	for {
		result, isOpen := <-r.ch
		if !isOpen {
			break
		}

		if result.err == nil {
			if showSummary {
				fmt.Printf("%s\n", result.summarize())
			}

			r.results = append(r.results, result)
		}

		r.wg.Done()
	}
}

func (r *requestResultset) wait() {
	r.wg.Wait()
	close(r.ch)
}

func newRequestResultSet(f requestMaker) *requestResultset {
	return &requestResultset{
		ch:       make(chan *requestResult, 1),
		executor: f,
		results:  make([]*requestResult, 0),
		wg:       new(sync.WaitGroup),
	}
}

type requestResult struct {
	err        error
	initialUrl string
	response   *http.Response
	timing     time.Duration
}

func (r *requestResult) getResourcesInResponse() []string {
	defer r.response.Body.Close()

	pageTokens := html.NewTokenizer(r.response.Body)
	finishedExaminingTokens := false
	bodyParsedResources := make([]string, 0)
	for !finishedExaminingTokens {
		if pageTokens.Next() == html.ErrorToken {
			finishedExaminingTokens = true
		}

		currentToken := pageTokens.Token()
		for _, a := range currentToken.Attr {
			if a.Key == "href" || a.Key == "src" {
				bodyParsedResources = append(bodyParsedResources, a.Val)
			}
		}
	}

	return bodyParsedResources
}

func (r *requestResult) getResponseStatus() int {
	if r.response != nil {
		return r.response.StatusCode
	}

	return -1
}

func (r *requestResult) summarize() string {
	contentType := " | "
	contentTypeHeader := r.response.Header.Get("Content-Type")
	if len(contentTypeHeader) > 0 {
		contentType = fmt.Sprintf(" | %s | ", strings.Split(contentTypeHeader, "; ")[0])
	}

	return fmt.Sprintf(
		"%s%s%d | %dms",
		r.initialUrl,
		contentType,
		r.response.StatusCode,
		r.timing.Milliseconds(),
	)
}

func makeRequest(c *http.Client, u string) *requestResult {
	localClient := *c
	result := &requestResult{
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

func setupHttpClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  true,
			IdleConnTimeout:    1 * time.Second,
			MaxIdleConns:       1,
		},
	}
}*/
