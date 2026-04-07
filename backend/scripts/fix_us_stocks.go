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

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	symbols := []string{"AAPL", "GOOG", "GOOGL", "MSFT", "TSLA", "AMZN", "META"}

	for _, s := range symbols {
		// 1. Delete ticks/candles
		_, _ = tx.Exec(`DELETE FROM stock_ticks WHERE symbol = $1`, s)
		_, _ = tx.Exec(`DELETE FROM stock_candles WHERE symbol = $1`, s)
		
		// 2. Delete from related tables by stock_id
		var stockID string
		err := tx.QueryRow(`SELECT stock_id FROM stock WHERE symbol = $1`, s).Scan(&stockID)
		if err == nil {
			_, _ = tx.Exec(`DELETE FROM stock_daily_data WHERE stock_id = $1`, stockID)
			_, _ = tx.Exec(`DELETE FROM orders WHERE stock_id = $1`, stockID)
			_, _ = tx.Exec(`DELETE FROM portfolio WHERE stock_id = $1`, stockID)
			_, _ = tx.Exec(`DELETE FROM watchlist WHERE stock_id = $1`, stockID)
			_, _ = tx.Exec(`DELETE FROM alerts WHERE stock_id = $1`, stockID)
			_, _ = tx.Exec(`DELETE FROM stock WHERE stock_id = $1`, stockID)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully removed all non-Indian stocks and their associated data.")
}
