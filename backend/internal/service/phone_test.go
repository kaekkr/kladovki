package service

import "testing"

func TestNormalizePhone(t *testing.T) {
	good := map[string]string{
		"+77019998877":        "+77019998877",
		"87019998877":         "+77019998877",
		"77019998877":         "+77019998877",
		" 8 (701) 999-88-77 ": "+77019998877",
		"+7 701 999 88 77":    "+77019998877",
	}
	for in, want := range good {
		if got, ok := NormalizePhone(in); !ok || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "900101300001", "a@b.kz", "superadmin:a@b.kz", "+1 202 555 0100",
		"7019998877", "+770199988770", "8701999887", "+8 701 999 88 77", "7+7019998877", "8701999887\x00", "٨٧٠١٩٩٩٨٨٧٧"} {
		if got, ok := NormalizePhone(in); ok {
			t.Errorf("NormalizePhone(%q) = %q, want rejected", in, got)
		}
	}
}
