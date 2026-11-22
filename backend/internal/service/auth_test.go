package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// The unknown-user path must cost a real bcrypt compare (timing must not reveal a registered ИИН).
func TestDummyHashHasRealCostAndMatchesNothing(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(dummyHash()))
	if err != nil || cost != bcrypt.DefaultCost {
		t.Fatalf("dummy hash cost = %d, %v; want %d", cost, err, bcrypt.DefaultCost)
	}
	if CheckPassword(dummyHash(), "password123") {
		t.Fatal("dummy hash must not match a password")
	}
}

func TestLoginKeySharesPhoneSpellings(t *testing.T) {
	if LoginKey("87001234567") != LoginKey(" +7 (700) 123-45-67 ") {
		t.Fatal("phone spellings must share one key")
	}
	if LoginKey("A@B.kz") != LoginKey("a@b.kz") {
		t.Fatal("email key is case-insensitive")
	}
	if LoginKey("900101300001") == LoginKey("+7900101300") {
		t.Fatal("different identifiers must not collide")
	}
}
