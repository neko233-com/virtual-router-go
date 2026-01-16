package VirtualRouterServer

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtSecret = "VirtualRouter-Center-Server-JWT-Secret-Key-Please-Change-In-Production-2024"

const tokenExpire = 24 * time.Hour
const refreshThreshold = 2 * time.Hour

func GenerateToken(userId string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userId,
		"iat": now.Unix(),
		"exp": now.Add(tokenExpire).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
}

func ValidateToken(tokenStr string) bool {
	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(jwtSecret), nil
	})
	return err == nil
}

func RefreshToken(tokenStr string) (string, bool) {
	claims, ok := parseClaims(tokenStr)
	if !ok {
		return "", false
	}
	userId, _ := claims["sub"].(string)
	if userId == "" {
		return "", false
	}
	newToken, err := GenerateToken(userId)
	if err != nil {
		return "", false
	}
	return newToken, true
}

func ShouldRefreshToken(tokenStr string) bool {
	claims, ok := parseClaims(tokenStr)
	if !ok {
		return true
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return true
	}
	remaining := time.Until(time.Unix(int64(exp), 0))
	return remaining < refreshThreshold
}

func GetTokenRemainingSeconds(tokenStr string) int64 {
	claims, ok := parseClaims(tokenStr)
	if !ok {
		return 0
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return 0
	}
	remaining := time.Until(time.Unix(int64(exp), 0))
	if remaining < 0 {
		return 0
	}
	return int64(remaining.Seconds())
}

func parseClaims(tokenStr string) (jwt.MapClaims, bool) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	return claims, ok
}
