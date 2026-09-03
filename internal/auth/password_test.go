package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}
	if err := VerifyPassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if err := VerifyPassword(hash, "wrong password"); err == nil {
		t.Fatal("wrong password was accepted")
	}
}
