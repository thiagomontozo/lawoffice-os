package auth

import (
	"bytes"
	"testing"
)

func TestPasswordHashAndVerification(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !CheckPassword(hash, "correct-horse-battery") {
		t.Fatal("valid password was rejected")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("invalid password was accepted")
	}
}
func TestPasswordPolicy(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password should be rejected")
	}
}
func TestSessionTokensAreRandomAndKeyed(t *testing.T) {
	tokenA, hashA, err := NewToken("secret-a")
	if err != nil {
		t.Fatal(err)
	}
	tokenB, hashB, err := NewToken("secret-a")
	if err != nil {
		t.Fatal(err)
	}
	if tokenA == tokenB || bytes.Equal(hashA, hashB) {
		t.Fatal("tokens must be unique")
	}
	if bytes.Equal(TokenHash(tokenA, "secret-a"), TokenHash(tokenA, "secret-b")) {
		t.Fatal("token hash must depend on server secret")
	}
}
