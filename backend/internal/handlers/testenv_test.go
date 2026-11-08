package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/database"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

// Integration tests run against a real Postgres.
// Start one with:
//
//	docker run -d --name kladovki_test_db -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test \
//	  -e POSTGRES_DB=kladovki_test -p 5442:5432 postgres:16-alpine
//	TEST_DATABASE_URL=postgres://test:test@localhost:5442/kladovki_test?sslmode=disable go test ./...
const webhookSecret = "test-webhook-secret"

type fakeApiPay struct {
	mu       sync.Mutex
	invoices []map[string]any
}

func (f *fakeApiPay) last() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.invoices) == 0 {
		return nil
	}
	return f.invoices[len(f.invoices)-1]
}

type testEnv struct {
	t      *testing.T
	db     *sql.DB
	router *gin.Engine
	tokens *service.TokenService
	apipay *fakeApiPay
	cfg    config.Config

	jkA, jkB           string
	adminA, adminB     string
	resident, other    string
	storageA, storage2 string // two storages in JK A
	storageB           string // storage in JK B
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	truncateAll(t, db)

	fake := &fakeApiPay{}
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fake.mu.Lock()
		fake.invoices = append(fake.invoices, body)
		fake.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1,"status":"processing"}`))
	}))
	t.Cleanup(apiSrv.Close)

	cfg := config.Config{
		Env:                 "production",
		APIPrefix:           "/api/v1",
		JWTSecret:           "test-jwt-secret-test-jwt-secret-32",
		JWTAccessTTL:        time.Hour,
		CookieName:          "access_token",
		ApiPayKey:           "test-key",
		ApiPayBaseURL:       apiSrv.URL,
		ApiPayWebhookSecret: webhookSecret,
	}

	svc := service.New(repository.New(db))
	tokens := service.NewTokenService(cfg.JWTSecret, cfg.JWTAccessTTL)
	r := gin.New()
	New(svc, tokens, cfg, middleware.NewAuth(tokens, cfg)).Register(r)

	env := &testEnv{t: t, db: db, router: r, tokens: tokens, apipay: fake, cfg: cfg}
	env.seed()
	return env
}

func truncateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		DO $$ DECLARE r RECORD;
		BEGIN
			FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func (e *testEnv) exec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.db.Exec(q, args...); err != nil {
		e.t.Fatalf("exec %q: %v", q, err)
	}
}

func (e *testEnv) seed() {
	now := time.Now().UTC()
	hash, _ := service.HashPassword("password123")
	phone := 0
	mkUser := func(role, iin string) string {
		id := uuid.NewString()
		phone++
		e.exec(`INSERT INTO users (id,role,full_name,phone,email,iin,password_hash,created_at)
			VALUES ($1,$2,'User',$3,$4,NULLIF($5,''),$6,$7)`,
			id, role, fmt.Sprintf("+7700000%04d", phone), id+"@x.kz", iin, hash, now)
		return id
	}
	e.adminA = mkUser("admin", "")
	e.adminB = mkUser("admin", "")
	e.resident = mkUser("resident", "900101300001")
	e.other = mkUser("resident", "900101300002")

	e.jkA, e.jkB = uuid.NewString(), uuid.NewString()
	for jk, owner := range map[string]string{e.jkA: e.adminA, e.jkB: e.adminB} {
		e.exec(`INSERT INTO jks (id,name,address,owner_id,created_at) VALUES ($1,$2,'Astana',$3,$4)`, jk, "JK "+jk[:4], owner, now)
		e.exec(`INSERT INTO tariffs (id,jk_id,amount,created_at) VALUES ($1,$2,1000,$3)`, uuid.NewString(), jk, now)
	}
	e.exec(`UPDATE users SET jk_id=$1 WHERE id IN ($2,$3,$4)`, e.jkA, e.adminA, e.resident, e.other)
	e.exec(`UPDATE users SET jk_id=$1 WHERE id=$2`, e.jkB, e.adminB)

	mkStorage := func(jk, number string) string {
		id := uuid.NewString()
		e.exec(`INSERT INTO storages (id,jk_id,number,area,floor,entrance,status,created_at)
			VALUES ($1,$2,$3,5,1,1,'free',$4)`, id, jk, number, now)
		return id
	}
	e.storageA = mkStorage(e.jkA, "1")
	e.storage2 = mkStorage(e.jkA, "2")
	e.storageB = mkStorage(e.jkB, "1")
}

func (e *testEnv) token(userID string) string {
	var role, jk string
	if err := e.db.QueryRow(`SELECT role, COALESCE(jk_id,'') FROM users WHERE id=$1`, userID).Scan(&role, &jk); err != nil {
		e.t.Fatalf("token lookup: %v", err)
	}
	tok, _, err := e.tokens.Generate(userID, role, jk, userID+"@x.kz")
	if err != nil {
		e.t.Fatalf("token: %v", err)
	}
	return tok
}

type resp struct {
	Code int
	Body []byte
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *testEnv) do(method, path, userID string, body any) resp {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, "/api/v1"+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("Authorization", "Bearer "+e.token(userID))
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Code: w.Code, Body: w.Body.Bytes()}
}

func sign(body []byte, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func (e *testEnv) webhook(payload any, signature string) resp {
	e.t.Helper()
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if signature == "valid" {
		signature = sign(b, webhookSecret)
	}
	if signature != "" {
		req.Header.Set("X-Webhook-Signature", signature)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Code: w.Code, Body: w.Body.Bytes()}
}

// lockAs books storage for `months` and returns the rental id.
func (e *testEnv) lockAs(userID, storageID string, months int) string {
	e.t.Helper()
	r := e.do(http.MethodPost, "/rentals/lock", userID, map[string]any{"storage_id": storageID, "months": months})
	if r.Code != http.StatusCreated {
		e.t.Fatalf("lock: %d %s", r.Code, r.Body)
	}
	var out struct {
		ID string `json:"id"`
	}
	r.json(e.t, &out)
	return out.ID
}

func (e *testEnv) paymentIDFor(rentalID string) string {
	var id string
	if err := e.db.QueryRowContext(context.Background(),
		`SELECT id FROM payments WHERE rental_id=$1 ORDER BY created_at DESC LIMIT 1`, rentalID).Scan(&id); err != nil {
		e.t.Fatalf("payment lookup: %v", err)
	}
	return id
}
