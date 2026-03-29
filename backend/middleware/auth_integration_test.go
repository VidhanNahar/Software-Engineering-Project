package middleware

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
    "time"

    "github.com/golang-jwt/jwt"
    "github.com/google/uuid"
)

// createTokenHelper creates a signed JWT token string for testing.
func createTokenHelper(userID uuid.UUID, secret string) string {
    claims := jwt.MapClaims{
        "userID": userID.String(),
        "exp":   time.Now().Add(time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    s, err := token.SignedString([]byte(secret))
    if err != nil {
        panic(err)
    }
    return s
}

func TestAuthMiddlewareIntegrationHeader(t *testing.T) {
    secret := "integration-test-secret"
    os.Setenv("JWT_SECRET", secret)
    defer os.Unsetenv("JWT_SECRET")

    // Setup handler with middleware
    handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
        if !ok {
            http.Error(w, "missing userID", http.StatusInternalServerError)
            return
        }
        json.NewEncoder(w).Encode(map[string]string{"user_id": userID.String()})
    }))

    req := httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Authorization", "Bearer "+createTokenHelper(uuid.New(), secret))
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)

    resp := w.Result()
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected status %d got %d", http.StatusOK, resp.StatusCode)
    }
    var data map[string]string
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }
    if _, ok := data["user_id"]; !ok {
        t.Fatal("expected user_id in response")
    }
}

func TestAuthMiddlewareIntegrationQuery(t *testing.T) {
    secret := "integration-test-secret"
    os.Setenv("JWT_SECRET", secret)
    defer os.Unsetenv("JWT_SECRET")

    handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
        if !ok {
            http.Error(w, "missing userID", http.StatusInternalServerError)
            return
        }
        json.NewEncoder(w).Encode(map[string]string{"user_id": userID.String()})
    }))

    token := createTokenHelper(uuid.New(), secret)
    req := httptest.NewRequest("GET", "/?token="+token, nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)

    resp := w.Result()
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected status %d got %d", http.StatusOK, resp.StatusCode)
    }
    var data map[string]string
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }
    if _, ok := data["user_id"]; !ok {
        t.Fatal("expected user_id in response")
    }
}

func TestAuthMiddlewareIntegrationMissing(t *testing.T) {
    // No token provided
    os.Setenv("JWT_SECRET", "integration-test-secret")
    defer os.Unsetenv("JWT_SECRET")

    handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "should not reach handler", http.StatusInternalServerError)
    }))

    req := httptest.NewRequest("GET", "/", nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)

    resp := w.Result()
    if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest {
        t.Fatalf("expected unauthorized or bad request got %d", resp.StatusCode)
    }
}
