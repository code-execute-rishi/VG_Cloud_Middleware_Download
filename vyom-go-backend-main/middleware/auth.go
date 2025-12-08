package middleware

import (
    "context"
    "net/http"
    "strings"
    "github.com/clerk/clerk-sdk-go/v2/jwt"
)

type contextKey string

const ClerkUserIDKey contextKey = "clerk_user_id"

func ClerkAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        
        claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
            Token: token,
        })
        if err != nil {
            http.Error(w, "Invalid token: " + err.Error(), http.StatusUnauthorized)
            return
        }
        
        ctx := context.WithValue(r.Context(), ClerkUserIDKey, claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}