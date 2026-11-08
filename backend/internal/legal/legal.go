// Package legal holds the user agreements written by the product owner
// (Telegram, 2026-09-21): one for residents, one for ОСИ, in Russian and Kazakh.
//
// To publish a new text: edit texts/<doc>.<lang>.txt and bump its version
// below. Residents then accept the new version at their next payment/renewal;
// ОСИ admins are asked again when they open the admin panel.
package legal

import (
	"embed"
	"strings"
)

//go:embed texts/*.txt
var files embed.FS

const (
	ResidentTerms = "resident_terms" // Пользовательское соглашение и правила аренды кладовых
	OSIRules      = "osi_rules"      // Регламент использования сервиса для ОСИ
)

var versions = map[string]string{
	ResidentTerms: "2026-09-21",
	OSIRules:      "2026-09-21",
}

// Document is one agreement in one language.
type Document struct {
	Doc     string `json:"doc"`
	Version string `json:"version"`
	Lang    string `json:"lang"`
	Title   string `json:"title"`
	Text    string `json:"text"`
}

// Version returns the current version of a document ("" if unknown).
func Version(doc string) string { return versions[doc] }

// Get returns a document; lang is "ru" (default) or "kk".
func Get(doc, lang string) (Document, bool) {
	if _, ok := versions[doc]; !ok {
		return Document{}, false
	}
	if lang != "kk" {
		lang = "ru"
	}
	b, err := files.ReadFile("texts/" + doc + "." + lang + ".txt")
	if err != nil {
		return Document{}, false
	}
	title, text, _ := strings.Cut(strings.TrimSpace(string(b)), "\n")
	return Document{Doc: doc, Version: versions[doc], Lang: lang, Title: strings.TrimSpace(title), Text: strings.TrimSpace(text)}, true
}
