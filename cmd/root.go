package cmd
import (
  "crypto/sha1"
  "crypto/tls"
  "fmt"
  "io"
  "net/http"
  "net/url"
  "os"
  "strconv"
  "strings"
  "sync"
  "time"

  "github.com/spf13/cobra"
  "golang.org/x/net/html"
)

type ResourceResult struct {
  RequestErr error
  RequestStatus int
  ResourceUrl string
  Timing time.Duration
}

var AnalyzeTls bool
var Insecure bool
var ShowHeaders bool
var rootCmd = &cobra.Command{
  Use: "rtime",
  Short: "rtime is for request timing and analysis",
  Long: `Command-line request timer and inspector.
  Makes request(s) to a page or a specific resource (endpoint). Get additional
  information such response headers or TLS connection data.`,
}
var endpointCmd = &cobra.Command{
  Use: "endpoint [url]",
  Short: "request a specific resource",
  Long: `Attempts to request the specified URL. Does not inspect the response
  body.`,
  RunE: func(cmd *cobra.Command, args []string) error {
    fmt.Println("not yet implemented")
    return nil
  },
}
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
      /*
      go func(resourceLoc string, hc *http.Client, ch chan ResourceResult, wg *sync.WaitGroup) {
        defer wg.Done()
        req, newRequestErr := http.NewRequest("GET", resourceLoc, nil)
        if newRequestErr != nil {
          fmt.Printf("%+v\n", newRequestErr)
          return
        }
        req.Close = true
        requestStart := time.Now()
        resp, respErr := hc.Do(req)
        currentResult := ResourceResult{
          RequestErr: respErr,
          ResourceUrl: resourceLoc,
          Timing: time.Now().Sub(requestStart),
        }
        if respErr == nil {
          currentResult.RequestStatus = resp.StatusCode
        }

        ch <- currentResult
      }(fmt.Sprintf("%s%s", formattedUrl.String(), resource), httpClient, timingCh, resourceWg)
      */
      go getResourceAsync(
        fmt.Sprintf("%s%s", formattedUrl.String(), resource),
        httpClient,
        timingCh,
        resourceWg,
      )
    }

    resourceWg.Wait()
    fmt.Printf("finished getting associated resources in %s\n", time.Now().Sub(resourcesRequestStart))
    close(timingCh)
    var largestResourceRequestTime time.Duration
    for _, t := range timings {
      // fmt.Printf("  a request finished at %s\n", rt)
      // fmt.Printf("  %d - %s %s\n", t.RequestStatus, t.ResourceUrl, t.Timing)
      if t.Timing > largestResourceRequestTime {
        largestResourceRequestTime = t.Timing
      }
    }

    fmt.Printf("longest associated resource request time: %s\n", largestResourceRequestTime)
    fmt.Printf("total request estimated at %s\n", intialRequestTime + largestResourceRequestTime)
    if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
      showResponseHeaders(resp)
    }

    if requestInsecure, _ := cmd.Flags().GetBool("analyze-tls"); requestInsecure {
      showResponseTlsInfo(resp)
    }

    return nil
  },
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}

func init() {
  rootCmd.AddCommand(pageCmd)
  rootCmd.AddCommand(endpointCmd)

  rootCmd.PersistentFlags().BoolVarP(&Insecure, "insecure", "", false, "make insecure request(s); sans https")
  rootCmd.PersistentFlags().BoolVarP(&AnalyzeTls, "analyze-tls", "", false, "show TLS information from response")
  rootCmd.PersistentFlags().BoolVarP(&ShowHeaders, "show-headers", "", false, "show headers from final request")
}

func formatUrl(host string, secure bool) string {
  formattedUrl, err := url.Parse(host)
  if err != nil || len(host) == 0 {
    //TODO: better logging
    //fmt.Printf("FormatUrl() - could not format: %s\n", host)
    return ""
  }

  if secure {
    formattedUrl.Scheme = "https"
  } else {
    formattedUrl.Scheme = "http"
  }

  return formattedUrl.String()
}

func getResourceAsync(loc string, hc *http.Client, ch chan ResourceResult, wg *sync.WaitGroup) {
  defer wg.Done()

  req, newRequestErr := http.NewRequest("GET", loc, nil)
  if newRequestErr != nil {
    // short circuit; incorrectly formated http.Request{}
    ch <- ResourceResult{ RequestErr: newRequestErr }
    return
  }

  req.Close = true
  requestStart := time.Now()
  resp, respErr := hc.Do(req)
  currentResult := ResourceResult{
    RequestErr: respErr,
    ResourceUrl: loc,
    Timing: time.Now().Sub(requestStart),
  }
  if respErr == nil {
    currentResult.RequestStatus = resp.StatusCode
  }

  ch <- currentResult
}
func getResourcesFromResponseBody(requestBody io.ReadCloser) []string {
  defer requestBody.Close()

  pageTokens := html.NewTokenizer(requestBody)
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
func showResponseHeaders(resp *http.Response) {
  fmt.Println()
  fmt.Printf("headers\n-------\n")
  for headerKey, headerValue := range resp.Header {
    fmt.Printf("%4s%s: %s\n", " ", headerKey, headerValue)
  }
  fmt.Println()
}
func showResponseTlsInfo(resp *http.Response) {
  var fingerprint strings.Builder
  tlsVersionsByCode := map[uint16]string{
    0x0300: "SSL 3.0",
    0x0301: "TLS 1.0",
    0x0302: "TLS 1.1",
    0x0303: "TLS 1.2",
    0x0304: "TLS 1.3",
  }

  fmt.Printf(
    "TLS Connection Info\n%s\ncipher suite:%27s\nversion:%17s\nassociated certs:%2d\n",
    "-------------------",
    tls.CipherSuiteName(resp.TLS.CipherSuite),
    tlsVersionsByCode[resp.TLS.Version],
    len(resp.TLS.PeerCertificates),
  )
  for _, cert := range resp.TLS.PeerCertificates {
    fpBytes := sha1.Sum(cert.Raw)
    for i:=0; i<len(fpBytes); i++ {
      fingerprint.WriteString(strconv.FormatInt(int64(fpBytes[i]), 16))
      if i < (len(fpBytes) - 1) {
        fingerprint.WriteString(":")
      }
    }

    fmt.Printf(
      "%4sCA? %t\n%4sexpires on: %s\n%4sissuer: %s\n%4ssubject: %s\n%4sfingerprint (sha1): %s\n\n",
      " ",
      cert.IsCA,
      " ",
      cert.NotAfter,
      " ",
      cert.Issuer.String(),
      " ",
      cert.Subject.String(),
      " ",
      fingerprint.String(),
    )
  }
}
