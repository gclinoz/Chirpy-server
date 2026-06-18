package auth

import (
	"testing"
	"time"
	"net/http"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	pswd := "onceUponAtime"

	hash, err := HashPassword(pswd)
	if err != nil {
		t.Errorf("Error when creating hashes: %v", err)
	}

	match, err := CheckPasswordHash(pswd, hash)
	if err != nil {
		t.Errorf("Something wrong during comparsion: %v", err)
	}
	if match != true {
		t.Errorf("Hashed string does not match")
	}

	match, err = CheckPasswordHash("wrongpassword", hash)
	if err != nil {
		t.Errorf("Something wrong during comparsion: %v", err)
	}
	if match == true {
		t.Errorf("Hashed string match on wrong password")
	}
}

func TestJWTValid (t *testing.T) {
	uid := uuid.New()
	secret := "onceUponAtime"

	s, err := MakeJWT(uid, secret, time.Hour)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	validID, err := ValidateJWT(s, secret)
	if err != nil {
		t.Fatalf("JWT validation fail: %s", err)
	}

	if uid != validID {
		t.Errorf("ID mismatch")
	}
}

func TestJWTExpired(t *testing.T) {
	uid := uuid.New()
	secret := "onceUponAtime"

	s, err := MakeJWT(uid, secret, -time.Second)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	_, err = ValidateJWT(s, secret)
	if err == nil {
		t.Fatalf("expected an error for expired token, got nil")
	}
}

func TestJWTWrong (t *testing.T) {
	uid := uuid.New()
	secret := "onceUponAtime"

	s, err := MakeJWT(uid, secret, time.Hour)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	_, err = ValidateJWT(s, "wrongsecret")
	if err == nil {
		t.Fatalf("expected an error for wrong secret, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	tok := "onceUponAtime"
	headers := http.Header{}
	headers.Set("Authorization", "Bearer " + tok)

	ts, err := GetBearerToken(headers)
	if err != nil {
		t.Errorf("error when getting token string: %v", err)
	}
	if ts != tok {
		t.Errorf("expect %s, get %s", tok, ts)
	}

	headers.Set("Authorization", "Bearer wrongtoken")
	ts, err = GetBearerToken(headers)
	if err != nil {
		t.Errorf("error when getting token string: %v", err)
	}
	if ts == tok {
		t.Errorf("expect different tokens, get same")
	}
}

func TestGetBearerTokenMiss(t *testing.T) {
	headers := http.Header{}

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf("expect no token found, get no error")
	}
}

func TestGetBearerTokenFmt(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Tiger onceUponAtime")

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf("expect format error, get no error")
	}
}
