package auth

import (
	"testing"

	"forum/internal/database"
)

func TestValidateUserCredentials(t *testing.T) {
	cases := []struct {
		email    string
		username string
		password string
		ok       bool
	}{
		{"user@example.com", "tester", "secret123", true},
		{"bad", "tester", "secret123", false},
		{"user@example.com", "x", "secret123", false},
		{"user@example.com", "tester", "123", false},
	}
	for i, c := range cases {
		err := ValidateUserCredentials(c.email, c.username, c.password)
		if c.ok && err != nil {
			t.Fatalf("case %d expected ok, got err: %v", i, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("case %d expected error, got nil", i)
		}
	}
}

func TestPasswordHashing(t *testing.T) {
	pwd := "super-secret"
	hash, err := database.HashPassword(pwd)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if err := database.CheckPasswordHash(hash, pwd); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if err := database.CheckPasswordHash(hash, "wrong"); err == nil {
		t.Fatalf("expected failure for wrong password")
	}
}
