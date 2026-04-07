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

	updates := map[string]string{
		"dhairya.r@ahduni.edu.in": "Dhairya",
		"drumil.b@ahduni.edu.in":  "Drumil Bhati",
	}

	for email, name := range updates {
		res, err := db.Exec(`UPDATE users SET name = $1 WHERE email_id = $2`, name, email)
		if err != nil {
			log.Printf("Failed to update %s: %v", email, err)
			continue
		}
		affected, _ := res.RowsAffected()
		if affected > 0 {
			fmt.Printf("Updated %s to name %s\n", email, name)
		}
	}
}
