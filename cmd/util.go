package main

import (
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

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
		for i := 0; i < len(fpBytes); i++ {
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
