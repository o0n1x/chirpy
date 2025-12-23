package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
)

func TestHash(t *testing.T) {
	cases := map[string]struct {
		password string
	}{
		"simple": {"password"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			hash, err := HashPassword(tc.password)
			if err != nil {
				t.Errorf("Failed to hash: %v\n", err)
				return
			}

			isequal, _ := argon2id.ComparePasswordAndHash(tc.password, hash)
			if !isequal {
				t.Error("Hash didnt not match password\n")
			}

		})
	}

}

func TestCheckPassword(t *testing.T) {
	cases := map[string]struct {
		password string
	}{
		"simple": {"password"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			hash, err := argon2id.CreateHash(tc.password, argon2id.DefaultParams)
			if err != nil {
				t.Errorf("Failed to hash: %v\n", err)
				return
			}

			isequalReal, _ := argon2id.ComparePasswordAndHash(tc.password, hash)
			isequal, _ := CheckPasswordHash(tc.password, hash)
			if isequal != isequalReal {
				t.Error("Hash Comparison Failed\n")
			}

		})
	}

}

func TestJWT(t *testing.T) {
	cases := map[string]struct {
		userID uuid.UUID
		secret string
	}{
		"simple":      {uuid.New(), "AssumeItsASecret"},
		"empty":       {uuid.New(), ""},
		"really long": {uuid.New(), "AssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecretAssumeItsASecret"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			token, err := MakeJWT(tc.userID, tc.secret, time.Minute)
			if err != nil {
				t.Errorf("Failed to generate token: %v\n", err)
				return
			}
			user, err := ValidateJWT(token, tc.secret)
			if err != nil {
				t.Errorf("Failed to validate token: %v\n", err)
				return
			}
			if user != tc.userID {
				t.Errorf("uuid \n%v\n does not equal Retrieved UUID \n%v\n", user, tc.userID)
				return
			}

		})
	}

}

func TestGetBearerToken(t *testing.T) {
	cases := map[string]struct {
		token      string
		cleantoken string
	}{
		"simple": {"Bearer token", "token"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			header := http.Header{}
			header.Add("Authorization", tc.token)
			cleantoken, err := GetBearerToken(header)
			if err != nil {
				t.Errorf("Failed to get token: %v\n", err)
				return
			}

			if cleantoken != tc.cleantoken {
				t.Errorf("cleantoken %v does not equal to real cleantoken %v\n", cleantoken, tc.cleantoken)
				return
			}

		})
	}

}
