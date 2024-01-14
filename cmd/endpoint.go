package main

import (
	"fmt"

	"github.com/rdenson/rtime/pkg/resource"
	"github.com/spf13/cobra"
)

var endpointCmd = &cobra.Command{
	Use:   "endpoint [url]",
	Short: "request a specific resource",
	Long: `Attempts to request the specified URL. Does not inspect the response
  body.`,
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
		if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
			showResponseHeaders(reqResult.Response)
		}

		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(reqResult.Response)
		}

		return nil
	},
}
