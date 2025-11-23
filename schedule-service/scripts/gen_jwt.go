package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Simple helper to generate a JWT compatible with middleware expectations.
// Usage (PowerShell): go run scripts/gen_jwt.go -sub usuario123 -role admin -email usuario@example.com -secret $env:JWT_SECRET
func main() {
	sub := flag.String("sub", "test-user", "Subject / user id")
	role := flag.String("role", "admin", "Role claim (admin|operador|viewer)")
	email := flag.String("email", "user@example.com", "Email claim")
	secret := flag.String("secret", os.Getenv("JWT_SECRET"), "JWT secret (defaults from env JWT_SECRET)")
	dur := flag.Duration("ttl", time.Hour*8, "Token validity duration")
	flag.Parse()

	if *secret == "" {
		fmt.Println("JWT secret required (pass with -secret or set JWT_SECRET env var)")
		os.Exit(1)
	}

	claims := jwt.MapClaims{
		"sub":   *sub,
		"role":  *role,
		"email": *email,
		"exp":   time.Now().Add(*dur).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(*secret))
	if err != nil {
		fmt.Println("Error signing token:", err)
		os.Exit(1)
	}
	fmt.Println(signed)
}
