package main

import (
	"backend-go/database"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load env from the parent directory
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Connect to database
	db, err := database.Connect(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		fmt.Println("Failed to connect to database: ", err)
		return
	}
	fmt.Println("Successfully connected to database")
	defer database.Close(db)

	// Create a redis client
	rdb, err := database.ConnectRedis()
	if err != nil {
		fmt.Println("Failed to connect to redis: ", err)
		return
	}
	fmt.Println("Successfully connected to redis")
	defer database.CloseRedis(rdb)

	// Create a http router
	r := mux.NewRouter()

	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
