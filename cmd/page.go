package cmd
import (
  "crypto/tls"
  "fmt"
  "net/http"
  "net/url"
  "sync"
  "time"

  "github.com/spf13/cobra"
)

var pageCmd = &cobra.Command{
  Use: "page [url]",
  Short: "request a page, just as you would in your browser",
  Long: `Attempts to request the specified URL and resolve any associated
  resources. eg. css, images, scripts, etc. Requests for additional
  resources are made concurrently.`,
  RunE: func(cmd *cobra.Command, args []string) error {
    var (
      insecureTransport *http.Transport = &http.Transport{
        DisableCompression: true,
        IdleConnTimeout: 30 * time.Second,
        MaxIdleConns: 10,
        TLSClientConfig: &tls.Config {
          InsecureSkipVerify: true,
        },
      }
      standardTransport *http.Transport = &http.Transport{
        DisableCompression: true,
        IdleConnTimeout: 30 * time.Second,
        MaxIdleConns: 10,
      }
    )

    httpClient := &http.Client{
      CheckRedirect: func(req *http.Request, via []*http.Request) error {
        fmt.Printf(
          "%4sredirect! got %d, now requesting: %s\n",
          " ",
          req.Response.StatusCode, req.URL.String(),
        )
        //just return the initial response
        // return http.ErrUseLastResponse
        return nil
      },
      Timeout: 30 * time.Second,
      Transport: standardTransport,
    }

    formattedUrl, urlParseErr := url.Parse(args[0])
    if urlParseErr != nil {
      return urlParseErr
    }

    formattedUrl.Scheme = "https"
    if requestInsecure, _ := cmd.Flags().GetBool("insecure"); requestInsecure {
      httpClient.Transport = insecureTransport
      formattedUrl.Scheme = "http"
    }

    fmt.Printf("initially requesting: %s\n", formattedUrl.String())
    req, _ := http.NewRequest("GET", formattedUrl.String(), nil)
    req.Close = true
    requestStart := time.Now()
    resp, respErr := httpClient.Do(req)
    intialRequestTime := time.Now().Sub(requestStart)
    fmt.Printf("request took: %s\n", intialRequestTime)
    fmt.Printf("status: %s\n", resp.Status)
    if respErr != nil {
      return respErr
    }

    resourcesToResolve := getResourcesFromResponseBody(resp.Body)
    fmt.Println("resolving resources...")
    timings := make([]ResourceResult, 0)
    timingCh := make(chan ResourceResult, 1)
    resourceWg := new(sync.WaitGroup)
    httpClient.CheckRedirect = nil
    go func() {
      for {
        reqResult, isOpen := <- timingCh
        if !isOpen { break }
        timings = append(timings, reqResult)
      }
    }()
    resourcesRequestStart := time.Now()
    for _, resource := range resourcesToResolve {
      resourceWg.Add(1)
      go getResourceAsync(
        fmt.Sprintf("%s%s", formattedUrl.String(), resource),
        httpClient,
        timingCh,
        resourceWg,
      )
    }

    resourceWg.Wait()
    fmt.Printf(
      "%4sfinished requesting %d resource(s) in %s\n",
      " ",
      len(timings),
      time.Now().Sub(resourcesRequestStart),
    )
    close(timingCh)
    var largestResourceRequestTime time.Duration
    for _, t := range timings {
      if t.Timing > largestResourceRequestTime {
        largestResourceRequestTime = t.Timing
      }
    }

    fmt.Printf("%4slongest associated resource request time: %s\n", " ", largestResourceRequestTime)
    fmt.Printf("total request estimated at %s\n", intialRequestTime + largestResourceRequestTime)

    if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
      showResponseHeaders(resp)
    }

    if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
      showResponseTlsInfo(resp)
    }

    if showResourceRequests, _ := cmd.Flags().GetBool("show-resource-requests"); showResourceRequests {
      fmt.Println()
      fmt.Println("resources parsed from initial request body:")
      for _, t := range timings {
        fmt.Printf("%5s %s %5d - %s\n", " ", t.Timing, t.RequestStatus, t.ResourceUrl)
      }

      fmt.Println()
    }

    return nil
  },
}
