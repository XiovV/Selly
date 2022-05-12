package jwt

import (
	"github.com/golang-jwt/jwt"
	"time"
)

func IsTokenExpired(token string) bool {
	tok, _ := jwt.Parse(token, nil)

	return getExp(tok) < time.Now().Unix()
}

func getExp(token *jwt.Token) int64 {
	return int64(token.Claims.(jwt.MapClaims)["exp"].(float64))
}