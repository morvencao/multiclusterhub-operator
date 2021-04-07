package main

import (
	"fmt"
	"net/http"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/healthz", Ping)
	http.HandleFunc("/readyz", Ping)
	http.ListenAndServe(":8000", nil)
}
