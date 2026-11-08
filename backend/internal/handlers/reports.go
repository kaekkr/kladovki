package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
)

// Reports for the ОСИ (regulation п. 5.1): income for any period, storage
// statuses with tenants, and the charges/debt ledger. CSV with a UTF-8 BOM and
// ";" separators opens correctly in Excel with a Russian locale.

var kzZone = time.FixedZone("UTC+5", 5*3600)

const dateLayout = "2006-01-02"

// parsePeriod reads ?from=YYYY-MM-DD&to=YYYY-MM-DD (inclusive days, Kazakhstan time).
// Missing values: from = first day of the current month, to = today.
func parsePeriod(c *gin.Context) (from, to time.Time, ok bool) {
	now := time.Now().In(kzZone)
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, kzZone)
	toDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, kzZone)
	if v := c.Query("from"); v != "" {
		d, err := time.ParseInLocation(dateLayout, v, kzZone)
		if err != nil {
			return from, to, false
		}
		from = d
	}
	if v := c.Query("to"); v != "" {
		d, err := time.ParseInLocation(dateLayout, v, kzZone)
		if err != nil {
			return from, to, false
		}
		toDay = d
	}
	to = toDay.AddDate(0, 0, 1) // exclusive end
	return from, to, !to.Before(from)
}

func writeCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	var buf bytes.Buffer
	buf.WriteString("\uFEFF")
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	w.UseCRLF = true
	_ = w.Write(header)
	_ = w.WriteAll(rows)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func kzDateTime(t time.Time) string { return t.In(kzZone).Format("02.01.2006 15:04") }
func kzDay(t time.Time) string      { return t.In(kzZone).Format("02.01.2006") }

var incomeKind = map[string]string{"booking": "Оплата аренды", "renewal": "Оплата продления", "refund": "Возврат"}
var storageStatusRU = map[string]string{"free": "Свободно", "locked": "Забронировано", "occupied": "Занято"}
var chargeStatusRU = map[string]string{"paid": "Оплачено", "overdue": "Просрочка", "ended": "Завершена"}

func (h *Handler) reportJK(c *gin.Context) (string, bool) {
	jkID := c.Param("jk_id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return "", false
	}
	return jkID, true
}

func (h *Handler) PaymentsReport(c *gin.Context) {
	jkID, ok := h.reportJK(c)
	if !ok {
		return
	}
	from, to, ok := parsePeriod(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный период (формат ГГГГ-ММ-ДД)"})
		return
	}
	list, err := h.svc.IncomeReport(c.Request.Context(), jkID, from, to)
	if err != nil {
		log.Printf("PaymentsReport: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сформировать отчёт"})
		return
	}
	rows := make([][]string, 0, len(list))
	for _, x := range list {
		number := ""
		if x.StorageNumber != "" {
			number = "№" + x.StorageNumber
		}
		rows = append(rows, []string{kzDateTime(x.At), incomeKind[x.Kind], number, x.UserName,
			x.CompanyName, x.CompanyBIN, x.Phone, strconv.FormatInt(x.Amount, 10)})
	}
	writeCSV(c, fmt.Sprintf("postupleniya_%s_%s.csv", from.Format(dateLayout), to.AddDate(0, 0, -1).Format(dateLayout)),
		[]string{"Дата", "Операция", "Кладовка", "Арендатор", "Компания", "БИН", "Телефон", "Сумма, ₸"}, rows)
}

func (h *Handler) StoragesReport(c *gin.Context) {
	jkID, ok := h.reportJK(c)
	if !ok {
		return
	}
	list, err := h.svc.StorageReport(c.Request.Context(), jkID)
	if err != nil {
		log.Printf("StoragesReport: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сформировать отчёт"})
		return
	}
	rows := make([][]string, 0, len(list))
	for _, x := range list {
		paidUntil := ""
		if x.PaidUntil != nil {
			paidUntil = kzDay(*x.PaidUntil)
		}
		rows = append(rows, []string{x.Number, strings.Replace(strconv.FormatFloat(x.Area, 'f', -1, 64), ".", ",", 1) /* Russian Excel decimal comma */, strconv.Itoa(x.Floor),
			strconv.Itoa(x.Entrance), storageStatusRU[x.Status], x.TenantName, x.CompanyName, x.Phone, paidUntil})
	}
	writeCSV(c, "kladovki_"+time.Now().In(kzZone).Format(dateLayout)+".csv",
		[]string{"Кладовка", "Площадь, м²", "Этаж", "Подъезд", "Статус", "Арендатор", "Компания", "Телефон", "Оплачено до"}, rows)
}

func (h *Handler) ChargesReport(c *gin.Context) {
	jkID, ok := h.reportJK(c)
	if !ok {
		return
	}
	list, err := h.svc.ListCharges(c.Request.Context(), jkID)
	if err != nil {
		log.Printf("ChargesReport: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сформировать отчёт"})
		return
	}
	rows := make([][]string, 0, len(list))
	for _, x := range list {
		rows = append(rows, []string{x.StorageNumber, x.UserName, x.CompanyName, x.UserPhone,
			kzDay(x.StartsAt), kzDay(x.EndsAt), strconv.FormatInt(x.PricePerMonth, 10),
			strconv.FormatInt(x.Paid, 10), strconv.Itoa(x.OverdueDays), strconv.FormatInt(x.Debt, 10), chargeStatusRU[string(x.Status)]})
	}
	writeCSV(c, "nachisleniya_"+time.Now().In(kzZone).Format(dateLayout)+".csv",
		[]string{"Кладовка", "Арендатор", "Компания", "Телефон", "Начало", "Оплачено до", "Цена в месяц, ₸",
			"Оплачено, ₸", "Дней просрочки", "Долг, ₸", "Статус"}, rows)
}
