package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend-go/middleware"
	"backend-go/store"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func TestPortfolioAPI_Integration(t *testing.T) {
	// Connecting to the real local database
	dsn := "postgres://myuser:mypassword@localhost:5432/project_db?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check if the database is actually reachable
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping integration test: Database not reachable at %s. Error: %v", dsn, err)
	}

	// Initialize the real store and handler
	realStore := store.NewStore(db, nil)
	handler := NewTradeHandler(realStore, nil)

	// Prepare Test Data: Generate unique UUIDs to avoid conflicts
	testUserID := uuid.New()
	testStockID := uuid.New()

	// Clean up test data after the test finishes
	defer func() {
		db.Exec("DELETE FROM portfolio WHERE user_id = $1", testUserID)
		db.Exec("DELETE FROM stock WHERE stock_id = $1", testStockID)
		db.Exec("DELETE FROM users WHERE user_id = $1", testUserID)
	}()

	// Insert a mock user into the database
	_, err = db.Exec(`
		INSERT INTO users (user_id, name, email_id, password, date_of_birth)
		VALUES ($1, 'Test User', 'testuser_portfolio@example.com', 'password', '1990-01-01')
	`, testUserID)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Insert a mock stock into the database with a current price of $150
	_, err = db.Exec(`
		INSERT INTO stock (stock_id, symbol, name, price, previous_close, open_price, day_high, day_low, close_price, total_traded_qty, total_trades, total_traded_value, timestamp)
		VALUES ($1, 'TESTAPL', 'Test Apple', 150.00, 140.00, 145.00, 155.00, 140.00, 150.00, 1000, 10, 150000, $2)
	`, testStockID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test stock: %v", err)
	}

	// Insert a mock portfolio entry for the user
	// User bought 10 shares at $100 each.
	_, err = db.Exec(`
		INSERT INTO portfolio (user_id, stock_id, quantity, price, transaction_time)
		VALUES ($1, $2, 10, 100.00, $3)
	`, testUserID, testStockID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test portfolio: %v", err)
	}

	// 2. Act: Create and Execute the API Request
	req, err := http.NewRequest("GET", "/api/portfolio", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Inject the test UserID into the request context (simulating the auth middleware)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, testUserID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GetPortfolio(rr, req)

	// 3. Assert: Verify the Response
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %v", rr.Code)
	}

	// Parse the JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Validate Count
	count, ok := response["count"].(float64)
	if !ok || count != 1 {
		t.Errorf("Expected count to be 1, got %v", response["count"])
	}

	// Validate Holdings Array
	holdings, ok := response["holdings"].([]interface{})
	if !ok || len(holdings) == 0 {
		t.Fatalf("Expected holdings array, got %v", response["holdings"])
	}

	firstHolding := holdings[0].(map[string]interface{})

	// Check mapping and mathematical logic
	// Quantity: 10
	if qty, ok := firstHolding["quantity"].(float64); !ok || qty != 10 {
		t.Errorf("Expected quantity 10, got %v", firstHolding["quantity"])
	}

	// Avg Buy Price: 100
	if avgBuyPrice, ok := firstHolding["avg_buy_price"].(float64); !ok || avgBuyPrice != 100 {
		t.Errorf("Expected avg_buy_price 100, got %v", firstHolding["avg_buy_price"])
	}

	// Current Price: 150
	if currentPrice, ok := firstHolding["current_price"].(float64); !ok || currentPrice != 150 {
		t.Errorf("Expected current_price 150, got %v", firstHolding["current_price"])
	}

	// Position Value (Quantity * Current Price = 10 * 150 = 1500)
	if positionValue, ok := firstHolding["position_value"].(float64); !ok || positionValue != 1500 {
		t.Errorf("Expected position_value 1500, got %v", firstHolding["position_value"])
	}
}
