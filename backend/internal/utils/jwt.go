package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"ledger-backend/internal/config"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	FamilyID *uint  `json:"family_id,omitempty"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID uint, familyID *uint, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		FamilyID: familyID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(config.AppConfig.JWT.AccessTokenHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    config.AppConfig.JWT.Issuer,
		},
	}

	token := jwt.NewWithClaims(signingMethod(), claims)
	return token.SignedString([]byte(config.AppConfig.JWT.Secret))
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != signingMethod().Alg() {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func NewRefreshToken() (string, string, error) {
	raw := make([]byte, 48)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func signingMethod() jwt.SigningMethod {
	switch strings.ToUpper(config.AppConfig.JWT.SigningMethod) {
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return jwt.SigningMethodHS256
	}
}
