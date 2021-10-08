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


var rootCmd = &cobra.Command{
  Use: "rtime",
  Short: "rtime is for request timing and analysis",
  Long: `Command-line request timer and inspector.
  Makes request(s) to a page or a specific resource (endpoint). Get additional
  information such response headers or TLS connection data.`,
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}

func init() {
  rootCmd.AddCommand(endpointCmd)
  rootCmd.AddCommand(pageCmd)
  rootCmd.AddCommand(versionCmd)

  rootCmd.PersistentFlags().Bool("insecure", false, "make insecure request(s); sans https")
  rootCmd.PersistentFlags().Bool("analyze-tls", false, "show TLS information from response")
  rootCmd.PersistentFlags().Bool("show-headers", false, "show headers from final request")
  pageCmd.Flags().Bool("show-resource-requests", false, "show request responses for associated resources")
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

  fmt.Println()
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
