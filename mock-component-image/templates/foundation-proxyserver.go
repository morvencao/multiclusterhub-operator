package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/healthz", Ping)

	server := &http.Server{
		Addr:      ":6443",
		TLSConfig: configProxyServerTLS(),
	}
	err := server.ListenAndServeTLS("localhost.crt", "localhost.key")
	if err != nil {
		fmt.Errorf("Listen server tls error: %+v", err)
	}
}

func configProxyServerTLS() *tls.Config {
	certFile := "/var/run/ocm-webhook/tls.crt"
	keyFile := "/var/run/ocm-webhook/tls.key"
	sCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		fmt.Errorf("error %v", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{sCert},
	}
}
