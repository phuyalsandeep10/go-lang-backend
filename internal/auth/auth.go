package auth

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   string `json:"user_id"`
    FullName string `json:"full_name"`
    Email    string `json:"email"`
    Phone    string `json:"phone"`
    jwt.RegisteredClaims
}

type TokenDetails struct {
    Token     string `json:"token"`
    ExpiresIn string `json:"expires_in"`
    TokenType string `json:"token_type"`
}

func GenerateJWT(userID, fullName, email, phone, secret string) (*TokenDetails, error) {
    if secret == "" {
        return nil, fmt.Errorf("secret key cannot be empty")
    }
    if userID == "" {
        return nil, fmt.Errorf("user ID cannot be empty")
    }

    expirationTime := time.Now().Add(24 * time.Hour)
    claims := &Claims{
        UserID:   userID,
        FullName: fullName,
        Email:    email,
        Phone:    phone,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expirationTime),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(secret))
    if err != nil {
        return nil, fmt.Errorf("failed to sign token: %v", err)
    }

    // Calculate expires_in in seconds
    expiresIn := int64(24 * time.Hour / time.Second) // 86400 seconds
    return &TokenDetails{
        Token:     tokenString,
        ExpiresIn: fmt.Sprintf("%d", expiresIn),
        TokenType: "Bearer",
    }, nil
}

func ValidateJWT(tokenString, secret string) (*Claims, error) {
    if secret == "" {
        return nil, fmt.Errorf("secret key cannot be empty")
    }
    if tokenString == "" {
        return nil, fmt.Errorf("token string cannot be empty")
    }

    claims := &Claims{}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to parse token: %v", err)
    }
    if !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }
    return claims, nil
}
