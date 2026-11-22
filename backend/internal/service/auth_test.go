package service

import (
	"strings"
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

func TestValidLoginIdentifier(t *testing.T) {
	cases := []struct {
		name  string
		login string
		want  bool
	}{
		{"iin", "900101300001", true},
		{"255 Cyrillic letters", strings.Repeat("я", 255), true},
		{"256 letters", strings.Repeat("я", 256), false},
		{"NUL", "a\x00b", false},
		{"invalid UTF-8", "a\xffb", false},
	}
	for _, c := range cases {
		if got := ValidLoginIdentifier(c.login); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// Limiter keys are bounded whatever the identifier: long logins can't blow up memory.
func TestLoginKeyIsBounded(t *testing.T) {
	if k := LoginKey(strings.Repeat("x", 255)); len(k) > 80 {
		t.Fatalf("key length %d, want bounded (hashed)", len(k))
	}
	if LoginKey("a@b.kz") == LoginKey("b@b.kz") {
		t.Fatal("different identifiers must not share a key")
	}
}
