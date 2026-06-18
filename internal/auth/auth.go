package auth

import (
	"time"
	"net/http"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

func HashPassword(password string) (string, error) {
	out, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return out, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	test, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	if test == false {
		return false, nil
	}
	return true, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	mySigningKey := []byte(tokenSecret)
	claims := jwt.RegisteredClaims{
		Issuer:		"chirpy-access",
		IssuedAt:	jwt.NewNumericDate(time.Now()),
		ExpiresAt:	jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:	userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		return "", err
	}
	return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.UUID{}, err
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	bear := headers.Get("Authorization")
	if bear == "" {
		return "", fmt.Errorf("no token found")
	}

	if strings.Split(bear, " ")[0] != "Bearer" {
		return "", fmt.Errorf("wrong header format")
	}

	ts := strings.TrimSpace(strings.TrimPrefix(bear, "Bearer"))
	return ts, nil
}
