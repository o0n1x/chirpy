package auth

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		log.Printf("Error verifying password: %v", err)
		return false, err
	}
	return match, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	header := headers.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("there is no Authorization header")
	}
	cutheader, ok := strings.CutPrefix(header, "ApiKey ")
	if !ok {
		return "", fmt.Errorf("invalid format")
	}
	return cutheader, nil
}
