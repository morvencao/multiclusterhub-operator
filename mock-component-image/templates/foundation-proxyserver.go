package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/healthz", Ping)

	server := &http.Server{
		Addr:      ":6443",
		TLSConfig: configTLS(),
	}
	err := server.ListenAndServeTLS("localhost.crt", "localhost.key")
	if err != nil {
		fmt.Errorf("Listen server tls error: %+v", err)
	}
}

func configTLS() *tls.Config {
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
