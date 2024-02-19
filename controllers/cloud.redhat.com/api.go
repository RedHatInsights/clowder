package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
)

func CreateAPIServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/config/", func(w http.ResponseWriter, _ *http.Request) {
		jsonString, _ := json.Marshal(clowderconfig.LoadedConfig)
		w.Header().Add(
			"Content-Type", "application/json",
		)
		fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdapps/present/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetPresentApps())
		fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdapps/managed/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetManagedApps())
		fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdenvs/present/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetPresentEnvs())
		fmt.Fprintf(w, "%s", jsonString)
	})

	mux.HandleFunc("/clowdenvs/managed/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add(
			"Content-Type", "application/json",
		)
		jsonString, _ := json.Marshal(GetManagedEnvs())
		fmt.Fprintf(w, "%s", jsonString)
	})

	srv := http.Server{
		Addr:              "127.0.0.1:2019",
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	return &srv
}
