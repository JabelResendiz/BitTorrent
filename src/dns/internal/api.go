//internal/api.go

package internal

import (
    "encoding/json"
    "net/http"
)

var apilog = NewLogger("API")

func Start(store *Store) {

	apilog.Info("Starting HTTP server on port 6969")

    http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
        apilog.Debug("Received /add request")
		var rec Record
        
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
            http.Error(w, "invalid request body", http.StatusBadRequest)
            apilog.Error("Failed to decode JSON: %v", err)
            return
        }
		
		// json.NewDecoder(r.Body).Decode(&rec)
        store.Add(rec)
        w.Write([]byte("ok"))
    })

    http.HandleFunc("/del", func(w http.ResponseWriter, r *http.Request) {
		apilog.Debug("Received /del request")
        var data map[string]string

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
            http.Error(w, "invalid request body", http.StatusBadRequest)
            apilog.Error("Failed to decode JSON: %v", err)
            return
        }

        // json.NewDecoder(r.Body).Decode(&data)

        store.Delete(data["name"])

        w.Write([]byte("deleted"))
    })

    http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		apilog.Debug("Received /list request")
        json.NewEncoder(w).Encode(store.List())
    })

	if err := http.ListenAndServe(":6969", nil); err != nil {
        apilog.Error("HTTP server failed: %v", err)
    }
    // http.ListenAndServe(":8080", nil)
}