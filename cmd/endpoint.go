package main

import (
	"fmt"

	"github.com/rdenson/rtime/pkg/resource"
	"github.com/spf13/cobra"
)

/*
	var endpointCmd = &cobra.Command{
		Use:   "endpoint [url]",
		Short: "request a specific resource",
		Long: `Attempts to request the specified URL. Does not inspect the response
	  body.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, newRequestErr := resource.NewRequest(
			// resource.OptionRequestUrl(args[0]),
			)
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
				showResponseHeaders(reqResult.Response.Header)
			}

			if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
				showResponseTlsInfo(reqResult.Response.TLS)
			}

			return nil
		},
	}
*/
var endpointCmd = &cobra.Command{
	Use:   "endpoint [url]",
	Short: "request a specific resource",
	Long: `Attempts to request the specified URL. Does not inspect the response
  body.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(res.GetTlsState())
		}

		return nil
	},
}
