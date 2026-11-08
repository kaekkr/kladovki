package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFiles_FirstFileWinsAndRealEnvWinsOverAll(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, ".env.local")
	base := filepath.Join(dir, ".env")
	_ = os.WriteFile(local, []byte("KLAD_A=local\nKLAD_B=\n"), 0o600)
	_ = os.WriteFile(base, []byte("KLAD_A=base\nKLAD_B=base\nKLAD_C=base\n"), 0o600)
	t.Setenv("KLAD_C", "real")
	os.Unsetenv("KLAD_A")
	os.Unsetenv("KLAD_B")
	t.Cleanup(func() { os.Unsetenv("KLAD_A"); os.Unsetenv("KLAD_B") })

	loaded := loadEnvFiles(filepath.Join(dir, "missing"), local, base)

	if len(loaded) != 2 {
		t.Fatalf("loaded = %v, want the two existing files", loaded)
	}
	for k, want := range map[string]string{"KLAD_A": "local", "KLAD_B": "base", "KLAD_C": "real"} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}
