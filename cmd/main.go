package main

import (
	"github.com/telq-org/sitemap-builder/pkg/logger"
	"github.com/telq-org/sitemap-builder/pkg/sitemap"
	"net/http"
)

func healthz() {
	http.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(nil)
		logger.Err(err)
	}))
	logger.Err(http.ListenAndServe("0.0.0.0:80", nil))
}

func main() {
	go healthz()
	logger.Must(sitemap.Build())
}
