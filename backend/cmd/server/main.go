package main

import (
	"crypto/rand"
	"database/sql"
	"drivehive-backend/internal/api"
	"drivehive-backend/internal/auth"
	"drivehive-backend/internal/database"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow standard dev origins and Tauri production origins
		switch origin {
		case "http://localhost:5173", "tauri://localhost", "http://tauri.localhost":
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		case "":
			// Fallback for non-browser clients or direct hits
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

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

	// Health check for Tauri sidecar discovery
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
	})

	// Authentication Handlers
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		res, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", creds.Username, string(hashed))
		if err != nil {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		// Auto-join the default global hive
		userID, _ := res.LastInsertId()
		err = database.AddUserToHive(db, "default-hive", int(userID))
		if err != nil {
			log.Printf("Warning: Failed to auto-join user %d to default hive: %v", userID, err)
		}

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var hash string
		var userID int
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", creds.Username).Scan(&userID, &hash)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Auth: User '%s' not found in the current database", creds.Username)
			} else {
				log.Printf("Auth: Database error for '%s': %v", creds.Username, err)
			}
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password)); err != nil {
			log.Printf("Auth: Password mismatch for user '%s'", creds.Username)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := auth.GenerateToken(creds.Username, userID)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":    token,
			"username": creds.Username,
			"user_id":  userID,
		})
	})

	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "Missing or malformed token", http.StatusUnauthorized)
			return
		}

		claims, err := auth.VerifyToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": claims.Username,
			"user_id":  claims.UserID,
		})
	})

	// Middleware for protected routes
	withAuth := func(next func(w http.ResponseWriter, r *http.Request, claims *auth.Claims)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
				return
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

		// Verify permissions
		role, err := database.GetUserRole(db, claims.UserID, req.HiveID)
		if err != nil || (role != "owner" && role != "admin") {
			log.Printf("Unauthorized channel creation attempt by user %d in hive %s", claims.UserID, req.HiveID)
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		err = database.CreateChannel(db, req.ID, req.HiveID, req.Name, req.Type)
		if err != nil {
			log.Printf("Error creating channel: %v", err)
			http.Error(w, "Failed to create channel", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))

	http.HandleFunc("/hives/join", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		if r.Method != http.MethodPost {
			return
		}
		var req struct {
			HiveID string `json:"hive_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		err := database.AddUserToHive(db, req.HiveID, claims.UserID)
		if err != nil {
			http.Error(w, "Failed to join hive", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	http.HandleFunc("/hives/invite/create", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		if r.Method != http.MethodPost {
			return
		}
		var req struct {
			HiveID string `json:"hive_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Generate random 8-char hex code
		b := make([]byte, 4)
		rand.Read(b)
		code := fmt.Sprintf("%x", b)

		err := database.CreateInvite(db, code, req.HiveID, claims.UserID)
		if err != nil {
			http.Error(w, "Failed to create invite", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"code": code})
	}))

	http.HandleFunc("/hives/invite/join", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		if r.Method != http.MethodPost {
			return
		}
		var req struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		hiveID, err := database.GetHiveIDByInvite(db, req.Code)
		if err != nil {
			http.Error(w, "Invalid or expired invite code", http.StatusNotFound)
			return
		}

		err = database.AddUserToHive(db, hiveID, claims.UserID)
		if err != nil {
			http.Error(w, "Failed to join hive", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"hive_id": hiveID})
	}))

	http.HandleFunc("/channels", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		hiveID := r.URL.Query().Get("hive_id")
		if hiveID == "" {
			http.Error(w, "Missing hive_id", http.StatusBadRequest)
			return
		}

		// Verify the user actually belongs to this hive
		member, err := database.IsUserInHive(db, claims.UserID, hiveID)
		if err != nil || !member {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		channels, err := database.GetHiveChannels(db, hiveID)
		if err != nil {
			http.Error(w, "Failed to fetch channels", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(channels)
	}))

	http.HandleFunc("/hives/members", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		hiveID := r.URL.Query().Get("hive_id")
		if hiveID == "" {
			http.Error(w, "Missing hive_id", http.StatusBadRequest)
			return
		}

		// Ensure requester is a member of the hive they are querying
		member, err := database.IsUserInHive(db, claims.UserID, hiveID)
		if err != nil || !member {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		users, err := database.GetHiveMembers(db, hiveID)
		if err != nil {
			http.Error(w, "Failed to fetch members", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(users)
	}))

	http.HandleFunc("/messages", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		roomID := r.URL.Query().Get("room_id")
		if roomID == "" {
			http.Error(w, "Missing room_id", http.StatusBadRequest)
			return
		}

		// Authorization check
		member, err := database.IsUserInChannel(db, claims.UserID, roomID)
		if err != nil || !member {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse Pagination params
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 50
		}

		var before time.Time
		if b := r.URL.Query().Get("before"); b != "" {
			if t, err := time.Parse(time.RFC3339, b); err == nil {
				before = t
			}
		}

		messages, err := database.GetRecentMessages(db, roomID, before, limit)
		if err != nil {
			log.Printf("Error fetching messages: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}))

	http.HandleFunc("/me", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		user, err := database.GetUser(db, claims.UserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)
	}))

	http.HandleFunc("/me/update", withAuth(func(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
		if r.Method != http.MethodPost {
			return
		}

		var req struct {
			AvatarURL string `json:"avatar_url"`
			Bio       string `json:"bio"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		err := database.UpdateUserProfile(db, claims.UserID, req.AvatarURL, req.Bio)
		if err != nil {
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})

	log.Printf("DriveHive server starting on :%d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), enableCORS(http.DefaultServeMux))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
