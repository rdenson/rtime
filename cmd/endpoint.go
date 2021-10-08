package cmd
import (
  "fmt"

  "github.com/spf13/cobra"
)

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
