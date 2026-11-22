package cadastral

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		" 21:319:021:848:8 ":      "21:319:021:848:8",
		"21 : 319:021:848:8:вп-2": "21:319:021:848:8:ВП-2",
		"21:319:021:848:8:":       "21:319:021:848:8",
		"":                        "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidComplex(t *testing.T) {
	valid := []string{"21:319:021:848:8", "21:319:021:848"}
	invalid := []string{"", "21", "21:319:abc:848:8", "21::021", "21:319:021:848:8:ВП-2x?"}
	for _, v := range valid {
		if !ValidComplex(v) {
			t.Errorf("ValidComplex(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if ValidComplex(v) {
			t.Errorf("ValidComplex(%q) = true, want false", v)
		}
	}
}

func TestSplitPremises(t *testing.T) {
	complexNo, unit, ok := SplitPremises("21:319:021:848:8:ВП-2")
	if !ok || complexNo != "21:319:021:848:8" || unit != "ВП-2" {
		t.Fatalf("got (%q, %q, %v)", complexNo, unit, ok)
	}
	complexNo, unit, ok = SplitPremises("21:319:021:848:8:154")
	if !ok || complexNo != "21:319:021:848:8" || unit != "154" {
		t.Fatalf("flat: got (%q, %q, %v)", complexNo, unit, ok)
	}
	for _, bad := range []string{"", "21:319", "21:319:021:848:8:", "abc:def:ВП-2", "21:319:021:848:8:ВП 2!"} {
		if _, _, ok := SplitPremises(bad); ok {
			t.Errorf("SplitPremises(%q) ok, want not ok", bad)
		}
	}
}

func TestUnitKind(t *testing.T) {
	if UnitKind("154") != "flat" {
		t.Error("digits → flat")
	}
	if UnitKind("ВП-2") != "commercial" {
		t.Error("ВП-2 → commercial (нежилое помещение)")
	}
}
