package internal

import (
	"encoding/json"
	"net"
	"net/http"
)

var apilog = NewLogger("API")

// jsonResponse is a helper for sending JSON responses
func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func Start(store *Store) {

	apilog.Info("Starting HTTP server on port 6969")

	mux := http.NewServeMux()

	// ============================
	//      POST /add
	// ============================
	mux.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		apilog.Debug("Received /add request")

		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		var rec Record

		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			apilog.Error("Failed to decode JSON: %v", err)
			jsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "invalid JSON body",
			})
			return
		}

		// --- VALIDATION ---
		if rec.Name == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "missing field 'name'",
			})
			return
		}

		// Normalize IPs: allow "ip": "A" OR "ips": ["A","B"]
		if len(rec.IPs) == 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "missing field 'ips' (array of IPs)",
			})
			return
		}

		// Validate each IP
		for _, ip := range rec.IPs {
			if net.ParseIP(ip) == nil {
				jsonResponse(w, http.StatusBadRequest, map[string]string{
					"error": "invalid IP address: " + ip,
				})
				return
			}
		}

		if rec.TTL <= 0 {
			rec.TTL = 60 // default TTL if user does not provide one
			apilog.Warn("TTL not provided for %s, defaulting to 60", rec.Name)
		}

		store.Add(rec)

		jsonResponse(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "record added",
		})
	})

	// ============================
	//      POST /del
	// ============================
	mux.HandleFunc("/del", func(w http.ResponseWriter, r *http.Request) {
		apilog.Debug("Received /del request")

		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		var data map[string]string

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			apilog.Error("Failed to decode JSON: %v", err)
			jsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "invalid JSON body",
			})
			return
		}

		name, ok := data["name"]
		if !ok || name == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "missing field 'name'",
			})
			return
		}

		store.Delete(name)

		jsonResponse(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "record deleted",
		})
	})

	// ============================
	//      GET /list
	// ============================
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		apilog.Debug("Received /list request")

		if r.Method != http.MethodGet {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		recs := store.List()
		jsonResponse(w, http.StatusOK, recs)
	})

	// start server
	if err := http.ListenAndServe(":6969", mux); err != nil {
		apilog.Error("HTTP server failed: %v", err)
	}
}

// {
//   "status": "ok",
//   "message": "record added"
// }

// [
//   { "name": "db.local", "ip": "10.0.0.20", "ttl": 60, "timestamp": "..." },
//   { "name": "api.local", "ip": "10.0.0.30", "ttl": 120, "timestamp": "..." }
// ]
