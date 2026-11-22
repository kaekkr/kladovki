// Package cadastral parses Kazakhstan cadastral numbers of a residential
// complex (ЖК) and of the premises inside it.
//
//	ЖК:        21:319:021:848:8
//	premises:  21:319:021:848:8:ВП-2   (last facet = flat or non-residential unit)
//
// A premises belongs to a ЖК when its number without the last facet equals
// the ЖК's number. This is how the app finds a resident's ЖК; the ownership
// itself is still confirmed by the ОСИ (later: egov via Smart Bridge).
package cadastral

import (
	"regexp"
	"strings"
)

var (
	facetDigits = regexp.MustCompile(`^\d+$`)
	// A unit is a flat number ("154") or a non-residential code ("ВП-2", "НП-14").
	facetUnit = regexp.MustCompile(`^[\p{L}\d]+(?:[-/][\p{L}\d]+)*$`)
)

// minComplexFacets: a ЖК number has at least region:district:quarter:parcel.
const minComplexFacets = 4

// Normalize removes spaces, a trailing ":" and upper-cases letters.
func Normalize(raw string) string {
	s := strings.Join(strings.Fields(raw), "")
	s = strings.TrimRight(s, ":")
	return strings.ToUpper(s)
}

// ValidComplex reports whether s (normalized) is a ЖК number: digit facets only.
func ValidComplex(s string) bool {
	facets := strings.Split(s, ":")
	if len(facets) < minComplexFacets {
		return false
	}
	for _, f := range facets {
		if !facetDigits.MatchString(f) {
			return false
		}
	}
	return true
}

// SplitPremises splits a normalized premises number into the ЖК number and
// the unit (last facet). ok is false when the number is malformed.
func SplitPremises(s string) (complexNo, unit string, ok bool) {
	i := strings.LastIndex(s, ":")
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	complexNo, unit = s[:i], s[i+1:]
	if !ValidComplex(complexNo) || !facetUnit.MatchString(unit) {
		return "", "", false
	}
	return complexNo, unit, true
}

// UnitKind guesses the premises kind from the unit: digits → flat,
// anything else (ВП-2, НП-14) → commercial (non-residential).
func UnitKind(unit string) string {
	if facetDigits.MatchString(unit) {
		return "flat"
	}
	return "commercial"
}
