package main

import (
	"backend-go/database"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	db, err := database.Connect(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer database.Close(db)

	rows, err := db.Query(`SELECT user_id, name, email_id FROM users`)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("Users in database:")
	for rows.Next() {
		var id, name, email string
		rows.Scan(&id, &name, &email)
		fmt.Printf("- %s: %s (%s)\n", id, name, email)
	}
}
