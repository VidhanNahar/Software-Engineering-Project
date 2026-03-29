package market

import (
	"backend-go/model"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// --- Fake Runner for Integration ---
// This acts as our "Database/Service" layer
type FakeRunner struct {
	Called bool
}

func (f *FakeRunner) IsMarketOpen() (bool, error) {
	return true, nil // Market is open for integration test
}

func (f *FakeRunner) SimulateTickCycle(ctx context.Context, now time.Time) ([]model.StockTick, error) {
	f.Called = true
	// Generate dummy ticks for integration
	return []model.StockTick{
		{Symbol: "AAPL", Price: 150.0},
		{Symbol: "GOOG", Price: 2800.0},
	}, nil
}

func (f *FakeRunner) GetStocks() ([]model.StockQuote, error) {
	return []model.StockQuote{
		{Symbol: "AAPL", Price: 150.0},
		{Symbol: "GOOG", Price: 2800.0},
	}, nil
}

func (f *FakeRunner) GetLatestCandlesForSymbols(symbols []string, timeframe string) ([]model.StockCandle, error) {
	return []model.StockCandle{
		{Symbol: "AAPL", Close: 150.0},
	}, nil
}

// --- TC-06: Integration Test between Engine and WebSocket Broadcaster ---
func TestIntegration_EngineToWebSocket(t *testing.T) {
	// 1. Setup the Broadcaster
	broadcaster := NewWebSocketBroadcaster()

	// 2. Setup a real HTTP test server to handle a real WebSocket connection
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade websocket: %v", err)
		}
		// Register the real websocket connection with our broadcaster
		broadcaster.AddClient(conn)
	}))
	defer server.Close()

	// 3. Connect a "Client" to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client failed to connect to WebSocket: %v", err)
	}
	defer clientConn.Close()

	// Wait a tiny bit for the server to register the client
	time.Sleep(50 * time.Millisecond)

	// 4. Setup the MarketEngine with the FakeRunner and the real Broadcaster
	runner := &FakeRunner{}
	engine := NewMarketEngine(runner, broadcaster, 1*time.Hour)

	// 5. Trigger a market cycle manually (simulating a ticker firing)
	engine.runCycle(time.Now())

	// 6. Verify the Client receives the messages over the network!
	// We expect 3 messages based on runCycle: Ticks, Snapshot, and Candles.
	expectedMessageTypes := map[string]bool{
		"stock_tick":      false,
		"stocks_snapshot": false,
		"candle_update":   false,
	}

	for i := 0; i < 3; i++ {
		// Set a read deadline so the test doesn't hang if it fails
		clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]interface{}
		err := clientConn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("Failed to read JSON from websocket on message %d: %v", i+1, err)
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			t.Fatalf("Received message without a 'type' field")
		}

		// Mark the message type as received
		if _, exists := expectedMessageTypes[msgType]; exists {
			expectedMessageTypes[msgType] = true
			t.Logf("✅ Successfully received integration message type: %s", msgType)
		}
	}

	// Verify all expected messages were received
	for msgType, received := range expectedMessageTypes {
		if !received {
			t.Errorf("Did not receive expected message type: %s", msgType)
		}
	}
}
