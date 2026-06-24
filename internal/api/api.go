package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"testcontainers/internal/store"
)

func NewRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_,_ = w.Write([]byte("OK"))
})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var user store.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if user.Name == "" || user.Email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and email are required"})
			return
		}
		
		id, err := store.InsertUser(r.Context(), db, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		user.ID = id
		writeJSON(w, http.StatusCreated, user)
	})

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
			return
		}
		user, err := store.GetUserByID(r.Context(), db, id)
		if errors.Is(err, store.ErrorNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, user)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}