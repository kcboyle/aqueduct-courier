package network

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func NewClient(skipTLSVerification bool, dialTimeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipTLSVerification,
				MinVersion:         tls.VersionTLS12,
			},
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}
}
