package main

import (
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func showResponseHeaders(headers http.Header) {
	fmt.Println()
	fmt.Printf("headers\n-------\n")
	for headerKey, headerValue := range headers {
		fmt.Printf("%4s%s: %s\n", " ", headerKey, headerValue)
	}
	fmt.Println()
}

func showResponseTlsInfo(cs *tls.ConnectionState) {
	var fingerprint strings.Builder
	tlsVersionsByCode := map[uint16]string{
		0x0300: "SSL 3.0",
		0x0301: "TLS 1.0",
		0x0302: "TLS 1.1",
		0x0303: "TLS 1.2",
		0x0304: "TLS 1.3",
	}

	sanVals := make([]string, 0)

	fmt.Println()
	fmt.Printf(
		"TLS Connection Info\n%s\ncipher suite:%27s\nversion:%17s\nassociated certs:%2d\n",
		"-------------------",
		tls.CipherSuiteName(cs.CipherSuite),
		tlsVersionsByCode[cs.Version],
		len(cs.PeerCertificates),
	)
	for _, cert := range cs.PeerCertificates {
		fpBytes := sha1.Sum(cert.Raw)
		for i := 0; i < len(fpBytes); i++ {
			fingerprint.WriteString(strconv.FormatInt(int64(fpBytes[i]), 16))
			if i < (len(fpBytes) - 1) {
				fingerprint.WriteString(":")
			}
		}

		// get SANs
		sanVals = append(sanVals, cert.DNSNames...)
		sanVals = append(sanVals, cert.EmailAddresses...)
		for _, v := range cert.IPAddresses {
			sanVals = append(sanVals, v.String())
		}
		for _, v := range cert.URIs {
			sanVals = append(sanVals, v.String())
		}

		fmt.Printf(
			"%4sCA? %t\n%4sexpires on: %s\n%4sissuer: %s\n%4ssubject: %s\n%4sfingerprint (sha1): %s\n",
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
		if len(sanVals) > 0 {
			fmt.Printf("%*sSANs:\n", 4, " ")
			for _, s := range sanVals {
				fmt.Printf("%*s%s\n", 6, " ", s)
			}
		}

		fmt.Println()
	}
}
