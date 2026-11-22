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

func TestResolveEnv_DefaultsToProductionAndRejectsUnknown(t *testing.T) {
	cases := []struct {
		appEnv, env string
		want        string
		wantErr     bool
	}{
		{"", "", "production", false},
		{"development", "", "development", false},
		{"production", "", "production", false},
		{"", "development", "development", false},
		{"production", "development", "production", false}, // APP_ENV wins
		{"dev", "", "", true},
		{"Production", "", "", true},
		{"staging", "", "", true},
		{"", "test", "", true},
	}
	for _, tc := range cases {
		got, err := resolveEnv(tc.appEnv, tc.env)
		if (err != nil) != tc.wantErr {
			t.Errorf("resolveEnv(%q,%q) err = %v, wantErr %v", tc.appEnv, tc.env, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("resolveEnv(%q,%q) = %q, want %q", tc.appEnv, tc.env, got, tc.want)
		}
	}
}

func TestResolveJWTSecret_FallbackOnlyInDevelopment(t *testing.T) {
	if s, err := resolveJWTSecret("development", ""); err != nil || s == "" {
		t.Fatalf("development without secret: %q, %v; want the dev fallback", s, err)
	}
	if _, err := resolveJWTSecret("production", ""); err == nil {
		t.Fatal("production without JWT_SECRET must fail")
	}
	if s, err := resolveJWTSecret("production", "real-secret"); err != nil || s != "real-secret" {
		t.Fatalf("production with secret: %q, %v", s, err)
	}
}

func TestResolveTrustedProxies(t *testing.T) {
	if got := resolveTrustedProxies("", false); len(got) != 0 {
		t.Fatalf("nothing set must trust nobody: %v", got)
	}
	got := resolveTrustedProxies(" 10.0.0.0/8 , ,192.0.2.1", false)
	if len(got) != 2 || got[0] != "10.0.0.0/8" || got[1] != "192.0.2.1" {
		t.Fatalf("list: %v", got)
	}
	withRender := resolveTrustedProxies("", true)
	if len(withRender) != len(privateNetworks) || withRender[0] != "10.0.0.0/8" {
		t.Fatalf("render: %v", withRender)
	}
}

func TestLoad_UnsetEnvMeansProductionWithSecureCookie(t *testing.T) {
	t.Chdir(t.TempDir()) // no .env files here
	t.Setenv("APP_ENV", "")
	t.Setenv("ENV", "")
	t.Setenv("JWT_SECRET", "a-real-secret-for-this-test-only-32")
	t.Setenv("COOKIE_SECURE", "")
	cfg := Load()
	if cfg.Env != "production" || !cfg.CookieSecure {
		t.Fatalf("Env = %q, CookieSecure = %v; want production + secure", cfg.Env, cfg.CookieSecure)
	}
}
