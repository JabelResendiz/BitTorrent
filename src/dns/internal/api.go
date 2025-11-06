package internal

import (
    "encoding/json"
    "net/http"
)

func Start(store *Store) {
    http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
        var rec Record
        json.NewDecoder(r.Body).Decode(&rec)
        store.Add(rec)
        w.Write([]byte("ok"))
    })

    http.HandleFunc("/del", func(w http.ResponseWriter, r *http.Request) {
        var data map[string]string
        json.NewDecoder(r.Body).Decode(&data)
        store.Delete(data["name"])
        w.Write([]byte("deleted"))
    })

    http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(store.List())
    })

    http.ListenAndServe(":8080", nil)
}
