package main

import (
	"github.com/leaq-ru/sitemap-builder/logger"
	"github.com/leaq-ru/sitemap-builder/sitemap"
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
