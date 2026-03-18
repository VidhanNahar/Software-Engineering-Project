package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Create a subrouter with /api prefix
	api := r.PathPrefix("/api").Subrouter()

	// Create an api to check health of server
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Listen on port 8080
	http.ListenAndServe(":8080", r)
}
