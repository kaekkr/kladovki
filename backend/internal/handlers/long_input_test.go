package handlers

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

// QA: a long Cyrillic termination comment was cut mid-character → 500.
func TestLongTerminationCommentIsCutByCharacters(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	comment := "a" + strings.Repeat("Я", 1500) // byte 1000 falls inside a letter
	r := e.do(http.MethodPost, "/rentals/"+id+"/terminate", e.resident, map[string]any{"comment": comment})
	if r.Code != http.StatusCreated {
		t.Fatalf("terminate: %d %s", r.Code, r.Body)
	}
	var stored string
	_ = e.db.QueryRow(`SELECT comment FROM requests WHERE rental_id=$1`, id).Scan(&stored)
	if !utf8.ValidString(stored) || utf8.RuneCountInString(stored) != 1000 {
		t.Fatalf("stored comment: %d runes, valid=%v", utf8.RuneCountInString(stored), utf8.ValidString(stored))
	}
}
