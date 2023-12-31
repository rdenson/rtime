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
		requestInsecure, _ := cmd.Flags().GetBool("insecure")
		req, newRequestErr := resource.NewRequest(args[0], !requestInsecure)
		if newRequestErr != nil {
			return newRequestErr
		}

		req.SetRedirectsToPrint()
		fmt.Printf("initially requesting: %s\n", req.Url)
		resp, reqResult := req.Exec()
		fmt.Printf("request took: %s\n", reqResult.Timing)
		if reqResult.RequestErr != nil {
			return reqResult.RequestErr
		}

		fmt.Printf("status: %s\n", resp.Status)
		if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
			showResponseHeaders(resp)
		}

		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(resp)
		}

		return nil
	},
}
