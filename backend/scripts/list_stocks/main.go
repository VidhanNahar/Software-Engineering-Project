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

	rows, err := db.Query(`SELECT symbol, name FROM stock ORDER BY symbol`)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("Current stocks in database:")
	for rows.Next() {
		var symbol, name string
		rows.Scan(&symbol, &name)
		fmt.Printf("- %s: %s\n", symbol, name)
	}
}
