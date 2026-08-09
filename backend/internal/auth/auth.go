package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(v string) (string, error) {
	if len([]rune(v)) < 10 || len([]byte(v)) > 72 {
		return "", errors.New("password must be 10 characters to 72 UTF-8 bytes")
	}
	b, e := bcrypt.GenerateFromPassword([]byte(v), bcrypt.DefaultCost)
	return string(b), e
}
func CheckPassword(hash, value string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(value)) == nil
}
func NewToken(secret string) (string, []byte, error) {
	raw := make([]byte, 32)
	if _, e := rand.Read(raw); e != nil {
		return "", nil, e
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, TokenHash(token, secret), nil
}
func TokenHash(token, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(token))
	return h.Sum(nil)
}
