package main

import (
	"database/sql"
	"drivehive-backend/internal/api"
	"drivehive-backend/internal/database"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	// Initialize SQLite database
	db, err := database.InitDB("./drivehive.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	hub := api.NewHub(db)
	go hub.Run()

	// Authentication Handlers
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)

		hashed, _ := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
		_, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", creds.Username, string(hashed))
		if err != nil {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)

		var hash string
		err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", creds.Username).Scan(&hash)
		if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password)) != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// For now, return a simple success. We will add JWT tokens in the next iteration.
		json.NewEncoder(w).Encode(map[string]string{"username": creds.Username})
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})

	log.Printf("DriveHive server starting on :%d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
