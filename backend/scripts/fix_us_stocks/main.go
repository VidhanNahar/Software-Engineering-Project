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

	// Find all non-Indian stocks (currency != INR or country != India)
	rows, err := tx.Query(`
		SELECT stock_id, symbol 
		FROM stock 
		WHERE currency_code <> 'INR' OR country <> 'India'
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	type StockRecord struct {
		ID     string
		Symbol string
	}

	var nonIndianStocks []StockRecord
	for rows.Next() {
		var stock StockRecord
		if err := rows.Scan(&stock.ID, &stock.Symbol); err != nil {
			log.Fatal(err)
		}
		nonIndianStocks = append(nonIndianStocks, stock)
	}

	if len(nonIndianStocks) == 0 {
		fmt.Println("No non-Indian stocks found in database.")
		tx.Rollback()
		return
	}

	fmt.Printf("Found %d non-Indian stock(s) to remove:\n", len(nonIndianStocks))
	for _, stock := range nonIndianStocks {
		fmt.Printf("  - %s (ID: %s)\n", stock.Symbol, stock.ID)
	}

	// Delete in order of dependencies
	for _, stock := range nonIndianStocks {
		// 1. Delete ticks/candles by symbol
		_, _ = tx.Exec(`DELETE FROM stock_ticks WHERE symbol = $1`, stock.Symbol)
		_, _ = tx.Exec(`DELETE FROM stock_candles WHERE symbol = $1`, stock.Symbol)

		// 2. Delete from related tables by stock_id
		_, _ = tx.Exec(`DELETE FROM stock_daily_data WHERE stock_id = $1`, stock.ID)
		_, _ = tx.Exec(`DELETE FROM orders WHERE stock_id = $1`, stock.ID)
		_, _ = tx.Exec(`DELETE FROM portfolio WHERE stock_id = $1`, stock.ID)
		_, _ = tx.Exec(`DELETE FROM watchlist WHERE stock_id = $1`, stock.ID)
		_, _ = tx.Exec(`DELETE FROM alerts WHERE stock_id = $1`, stock.ID)

		// 3. Finally delete the stock itself
		_, _ = tx.Exec(`DELETE FROM stock WHERE stock_id = $1`, stock.ID)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully removed all non-Indian stocks and their associated data.")
}
