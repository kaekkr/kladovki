package legal

import (
	"strings"
	"testing"
)

func TestEveryDocumentExistsInBothLanguages(t *testing.T) {
	for _, doc := range []string{ResidentTerms, OSIRules} {
		if Version(doc) == "" {
			t.Fatalf("%s has no version", doc)
		}
		for _, lang := range []string{"ru", "kk"} {
			d, ok := Get(doc, lang)
			if !ok || d.Title == "" || len(d.Text) < 1000 || d.Lang != lang {
				t.Fatalf("%s/%s: ok=%v title=%q len=%d", doc, lang, ok, d.Title, len(d.Text))
			}
			if strings.ContainsRune(d.Text, '​') {
				t.Fatalf("%s/%s contains zero-width spaces", doc, lang)
			}
		}
	}
	if d, _ := Get(ResidentTerms, "xx"); d.Lang != "ru" {
		t.Fatal("unknown language must fall back to Russian")
	}
	if _, ok := Get("nope", "ru"); ok {
		t.Fatal("unknown document")
	}
	if d, _ := Get(ResidentTerms, "ru"); !strings.Contains(d.Text, "исключительно собственникам") {
		t.Fatal("resident terms must contain п. 2.1")
	}
}
