// Package controllers provides the main controller implementations for Clowder resources
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
)

// CreateAPIServer creates and configures the HTTP API server for Clowder configuration endpoints
func CreateAPIServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/config/", func(w http.ResponseWriter, _ *http.Request) {
		jsonString, _ := json.Marshal(clowderconfig.LoadedConfig)
		w.Header().Add(
			"Content-Type", "application/json",
		)
		_, _ = fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdapps/present/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetPresentApps())
		_, _ = fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdapps/managed/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetManagedApps())
		_, _ = fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdenvs/present/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetPresentEnvs())
		_, _ = fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdenvs/managed/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetManagedEnvs())
		_, _ = fmt.Fprintf(w, "%s", jsonString)
	})

	srv := http.Server{
		Addr:              "127.0.0.1:2019",
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	return &srv
}
