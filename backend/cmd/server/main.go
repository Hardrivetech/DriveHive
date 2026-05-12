package main

import (
	"database/sql"
	"drivehive-backend/internal/api"
	"drivehive-backend/internal/auth"
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
	dbPath := flag.String("db", "./drivehive.db", "Path to SQLite database")
	flag.Parse()

	// Initialize SQLite database
	db, err := database.InitDB(*dbPath)
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
		var userID int
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", creds.Username).Scan(&userID, &hash)
		if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password)) != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := auth.GenerateToken(creds.Username, userID)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"token":    token,
			"username": creds.Username,
			"user_id":  fmt.Sprintf("%d", userID),
		})
	})

	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		claims, err := auth.VerifyToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"username": claims.Username})
	})

	// Middleware for protected routes
	withAuth := func(next func(w http.ResponseWriter, r *http.Request, claims *auth.Claims)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			claims, err := auth.VerifyToken(token)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r, claims)
		}
	}

	// Hive Management
	http.HandleFunc("/hives/create", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		if r.Method != http.MethodPost {
			return
		}

		var req struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		err = database.CreateHive(db, req.ID, req.Name, claims.UserID)
		if err != nil {
			log.Printf("Error creating hive: %v", err)
			http.Error(w, "Failed to create hive", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))

	http.HandleFunc("/hives", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		hives, err := database.GetUserHives(db, claims.UserID)
		if err != nil {
			http.Error(w, "Failed to fetch hives", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(hives)
	}))

	http.HandleFunc("/channels/create", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		if r.Method != http.MethodPost {
			return
		}

		var req struct {
			ID     string `json:"id"`
			HiveID string `json:"hive_id"`
			Name   string `json:"name"`
			Type   string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Note: In a production app, verify the user is the owner/admin of the Hive before creating
		err = database.CreateChannel(db, req.ID, req.HiveID, req.Name, req.Type)
		if err != nil {
			log.Printf("Error creating channel: %v", err)
			http.Error(w, "Failed to create channel", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))

	http.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		hiveID := r.URL.Query().Get("hive_id")
		if hiveID == "" {
			http.Error(w, "Missing hive_id", http.StatusBadRequest)
			return
		}
		channels, err := database.GetHiveChannels(db, hiveID)
		if err != nil {
			http.Error(w, "Failed to fetch channels", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(channels)
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
