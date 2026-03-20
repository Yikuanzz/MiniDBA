package auth

import (
	"testing"
	"time"
)

func TestSecretMatch(t *testing.T) {
	if !SecretMatch("a", "a") {
		t.Fatal()
	}
	if SecretMatch("a", "b") {
		t.Fatal()
	}
}

func TestSignVerifyToken(t *testing.T) {
	const sec = "test-secret"
	tok, err := SignToken(sec, time.Now().Add(time.Hour).Unix())
	if err != nil || tok == "" {
		t.Fatal(err)
	}
	if !VerifyToken(sec, tok) {
		t.Fatal("verify")
	}
	if VerifyToken("other", tok) {
		t.Fatal("wrong secret")
	}
}
