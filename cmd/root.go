package cmd
import (
  "fmt"
  "os"
  "time"

  "github.com/spf13/cobra"
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
  pageCmd.Flags().Bool("show-resources-requested", false, "show request responses for associated resources")
}
