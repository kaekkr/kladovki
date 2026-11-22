package handlers

import "testing"

// QA: a tenant name like =HYPERLINK(...) ran as a formula when the ОСИ opened the CSV.
func TestCSVCellNeutralisesFormulas(t *testing.T) {
	cases := map[string]string{
		`=HYPERLINK("http://x","y")`: `'=HYPERLINK("http://x","y")`,
		"+cmd|' /C calc'!A0":         "'+cmd|' /C calc'!A0",
		"-1+1":                       "'-1+1",
		"@SUM(A1:A2)":                "'@SUM(A1:A2)",
		"\tTAB":                      "'\tTAB",
		"\rCR":                       "'\rCR",
		"Ержан":                      "Ержан",
		"":                           "",
		"5000":                       "5000",
		"-2500":                      "-2500", // a plain number stays a number
		"+77001234567":               "+77001234567",
		"12,5":                       "12,5",
	}
	for in, want := range cases {
		if got := csvCell(in); got != want {
			t.Errorf("csvCell(%q) = %q, want %q", in, got, want)
		}
	}
}
