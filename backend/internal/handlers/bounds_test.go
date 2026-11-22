package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// QA: a huge tariff gave 500 but was saved, then price × months overflowed.
func TestTariffBounds(t *testing.T) {
	e := newTestEnv(t)
	path := "/jks/" + e.jkA + "/tariff"
	for _, bad := range []int64{0, -5, 1_000_001, 9223372036854775807} {
		if r := e.do(http.MethodPut, path, e.adminA, map[string]any{"amount": bad}); r.Code != http.StatusBadRequest {
			t.Fatalf("tariff %d: %d %s, want 400", bad, r.Code, r.Body)
		}
		if r := e.do(http.MethodGet, fmt.Sprintf("%s/preview?amount=%d", path, bad), e.adminA, nil); r.Code != http.StatusBadRequest {
			t.Fatalf("preview %d: %d, want 400", bad, r.Code)
		}
	}
	var amount int64
	_ = e.db.QueryRow(`SELECT amount FROM tariffs WHERE jk_id=$1`, e.jkA).Scan(&amount)
	if amount != 1000 {
		t.Fatalf("tariff = %d, a refused value must not be saved", amount)
	}
	if n := e.count(`SELECT count(*) FROM tariff_history WHERE jk_id=$1`, e.jkA); n != 0 {
		t.Fatalf("history rows = %d, want 0", n)
	}
	if r := e.do(http.MethodPut, path, e.adminA, map[string]any{"amount": 1_000_000}); r.Code != http.StatusOK {
		t.Fatalf("max tariff: %d %s", r.Code, r.Body)
	}
	// 5 m² × 1 000 000 × 120 months fits; the booking total must be positive.
	r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storageA, "months": 120})
	var rt struct {
		Price int64 `json:"price_per_month"`
	}
	r.json(t, &rt)
	if r.Code != http.StatusCreated || rt.Price != 5_000_000 {
		t.Fatalf("lock at max tariff: %d %s", r.Code, r.Body)
	}
}

func TestStorageBounds(t *testing.T) {
	e := newTestEnv(t)
	valid := map[string]any{"number": "100", "area": 2.5, "floor": -1, "entrance": 1}
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for kk, vv := range valid {
			m[kk] = vv
		}
		m[k] = v
		return m
	}
	bad := []map[string]any{
		with("area", 0.0001), // rounds to 0.00 → price 0
		with("area", 10000.01),
		with("area", 1e300),
		with("floor", -11),
		with("floor", 201),
		with("floor", -99999999999),
		with("entrance", -1),
		with("entrance", 101),
		with("number", strings.Repeat("Я", 51)),
	}
	for _, body := range bad {
		if r := e.do(http.MethodPost, "/storages", e.adminA, body); r.Code != http.StatusBadRequest {
			t.Fatalf("create %v: %d %s, want 400", body, r.Code, r.Body)
		}
		if r := e.do(http.MethodPut, "/storages/"+e.storageA, e.adminA, body); r.Code != http.StatusBadRequest {
			t.Fatalf("update %v: %d %s, want 400", body, r.Code, r.Body)
		}
	}
	r := e.do(http.MethodPost, "/storages", e.adminA, with("area", 0.014))
	var st struct {
		Area float64 `json:"area"`
	}
	r.json(t, &st)
	if r.Code != http.StatusCreated || st.Area != 0.01 {
		t.Fatalf("create 0.014 m²: %d %s, want 201 with area 0.01", r.Code, r.Body)
	}
}
