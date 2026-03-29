package middleware

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/golang-jwt/jwt"
    "github.com/google/uuid"
)

// helper to create a token string signed with the secret
func createToken(userID uuid.UUID, secret string) string {
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

// dummy handler to capture user id from context
func dummyHandler(w http.ResponseWriter, r *http.Request) {
    if v := r.Context().Value(UserIDKey); v != nil {
        json.NewEncoder(w).Encode(map[string]string{"user_id": v.(uuid.UUID).String()})
    } else {
        http.Error(w, "no user_id", http.StatusInternalServerError)
    }
}

func TestAuthMiddleware(t *testing.T) {
    secret := "test-secret"
    os.Setenv("JWT_SECRET", secret)
    defer os.Unsetenv("JWT_SECRET")

    tests := []struct {
        name             string
        header           string
        queryToken       string
        expectStatus     int
        expectBody       string
        expectHasUserID  bool
    }{
        {name: "missing token", header: "", queryToken: "", expectStatus: http.StatusUnauthorized, expectBody: "Missing authorization token", expectHasUserID: false},
        {name: "invalid header format", header: "Bearere abc", queryToken: "", expectStatus: http.StatusBadRequest, expectBody: "Invalid authorization header format", expectHasUserID: false},
        {name: "invalid token", header: "Bearer invalidtoken", queryToken: "", expectStatus: http.StatusBadRequest, expectBody: "Invalid or expired token", expectHasUserID: false},
        {name: "valid token header", header: "Bearer " + createToken(uuid.New(), secret), queryToken: "", expectStatus: http.StatusOK, expectHasUserID: true},
        {name: "valid token query", header: "", queryToken: createToken(uuid.New(), secret), expectStatus: http.StatusOK, expectHasUserID: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/", nil)
            if tt.header != "" {
                req.Header.Set("Authorization", tt.header)
            }
            if tt.queryToken != "" {
                q := req.URL.Query()
                q.Set("token", tt.queryToken)
                req.URL.RawQuery = q.Encode()
            }

            w := httptest.NewRecorder()
            handler := AuthMiddleware(http.HandlerFunc(dummyHandler))
            handler.ServeHTTP(w, req)

            resp := w.Result()
            if resp.StatusCode != tt.expectStatus {
                t.Fatalf("expected status %d got %d", tt.expectStatus, resp.StatusCode)
            }
            body := w.Body.String()
            if tt.expectHasUserID {
                var data map[string]string
                if err := json.Unmarshal([]byte(body), &data); err != nil {
                    t.Fatalf("expected JSON body, got %s", body)
                }
                if _, ok := data["user_id"]; !ok {
                    t.Fatalf("expected user_id in body: %s", body)
                }
            } else {
                if !contains(body, tt.expectBody) {
                    t.Fatalf("expected body to contain %q, got %q", tt.expectBody, body)
                }
            }
        })
    }
}

func contains(s, substr string) bool {
    return strings.Contains(s, substr)
}
