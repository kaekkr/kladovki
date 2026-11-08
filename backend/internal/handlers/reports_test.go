package handlers

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"
	"time"
)

var kz = time.FixedZone("UTC+5", 5*3600)

func (e *testEnv) csvReport(userID, path string) (resp, [][]string) {
	e.t.Helper()
	r := e.do(http.MethodGet, path, userID, nil)
	if r.Code != http.StatusOK {
		return r, nil
	}
	body := string(r.Body)
	if !strings.HasPrefix(body, "\uFEFF") {
		e.t.Fatalf("%s: CSV must start with a UTF-8 BOM so Excel shows Cyrillic", path)
	}
	rd := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\uFEFF")))
	rd.Comma = ';'
	rows, err := rd.ReadAll()
	if err != nil {
		e.t.Fatalf("%s: bad CSV: %v", path, err)
	}
	return r, rows
}

func cellsContain(rows [][]string, want ...string) bool {
	for _, row := range rows {
		line := strings.Join(row, "|")
		ok := true
		for _, w := range want {
			if !strings.Contains(line, w) {
				ok = false
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// ОСИ regulation п. 5.1: income for any period, with tenants' names / company names.
func TestIncomeReportForPeriod(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE users SET owner_type='company', company_name='ТОО Аптека', company_bin='180940012345' WHERE id=$1`, e.resident)
	recent := e.activeRental(e.resident, e.storageA) // 10 000 today
	old := e.activeRental(e.other, e.storage2)
	e.exec(`UPDATE payments SET created_at = now() - interval '40 days' WHERE rental_id=$1`, old)
	// A refund paid out today shows as a negative line.
	e.do(http.MethodPost, "/rentals/"+recent+"/terminate", e.resident, map[string]any{"comment": "x"})
	for _, q := range e.adminRequests(e.adminA) {
		e.do(http.MethodPatch, "/requests/"+q.ID+"/resolve", e.adminA, map[string]any{"action": "approve"})
	}

	from := time.Now().In(kz).AddDate(0, 0, -10).Format("2006-01-02")
	to := time.Now().In(kz).Format("2006-01-02")
	r, rows := e.csvReport(e.adminA, "/reports/jk/"+e.jkA+"/payments.csv?from="+from+"&to="+to)
	if r.Code != http.StatusOK {
		t.Fatalf("report: %d %s", r.Code, r.Body)
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("content-type %q", ct)
	}
	if !strings.Contains(r.Header.Get("Content-Disposition"), "attachment") {
		t.Fatal("must download as a file")
	}
	if rows[0][0] != "Дата" {
		t.Fatalf("header = %v", rows[0])
	}
	if !cellsContain(rows, "ТОО Аптека", "180940012345", "10000", "Оплата") {
		t.Fatalf("payment row missing: %v", rows)
	}
	if !cellsContain(rows, "-5000", "Возврат") {
		t.Fatalf("refund row missing: %v", rows)
	}
	if cellsContain(rows, "№2") || len(rows) != 3 { // header + payment + refund; the 40-day-old payment is outside
		t.Fatalf("period filter wrong: %v", rows)
	}

	// The JSON list used by the Платежи page accepts the same period.
	var list []map[string]any
	e.do(http.MethodGet, "/payments/jk/"+e.jkA+"?from="+from+"&to="+to, e.adminA, nil).json(t, &list)
	if len(list) != 1 || list[0]["company_name"] != "ТОО Аптека" {
		t.Fatalf("payments JSON = %v", list)
	}
}

func TestStorageAndChargesReports(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '30 days' + interval '1 hour' WHERE id=$1`, id)

	_, rows := e.csvReport(e.adminA, "/reports/jk/"+e.jkA+"/storages.csv")
	if !cellsContain(rows, "1", "Занято", "User") || !cellsContain(rows, "2", "Свободно") {
		t.Fatalf("storages report = %v", rows)
	}
	_, rows = e.csvReport(e.adminA, "/reports/jk/"+e.jkA+"/charges.csv")
	if !cellsContain(rows, "1", "5000", "30") {
		t.Fatalf("charges report = %v", rows)
	}

	for _, path := range []string{"payments.csv", "storages.csv", "charges.csv"} {
		if r := e.do(http.MethodGet, "/reports/jk/"+e.jkA+"/"+path, e.adminB, nil); r.Code != http.StatusForbidden {
			t.Fatalf("%s other ЖК admin: %d", path, r.Code)
		}
		if r := e.do(http.MethodGet, "/reports/jk/"+e.jkA+"/"+path, e.resident, nil); r.Code != http.StatusForbidden {
			t.Fatalf("%s resident: %d", path, r.Code)
		}
	}
	if r := e.do(http.MethodGet, "/reports/jk/"+e.jkA+"/payments.csv?from=2026-13-01", e.adminA, nil); r.Code != http.StatusBadRequest {
		t.Fatalf("bad date: %d", r.Code)
	}
}
