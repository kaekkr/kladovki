package service

import (
	"math"

	"github.com/kaekkr/kladovki/internal/models"
)

// CalcPrice считает стоимость аренды.
// Тариф = ₸ (int64) за 1 м² в месяц.
// Пример: 15000 ₸ * 5.5 м² * 3 мес = 247 500 ₸
func CalcPrice(area float64, tariffPerM2 int64, months int) models.PriceQuote {
	if months < 1 {
		months = 1
	}

	// Округляем до ближайшего целого тенге
	perMonth := int64(math.Round(area * float64(tariffPerM2)))
	total := perMonth * int64(months)

	return models.PriceQuote{
		Area:          area,
		TariffPerM2:   tariffPerM2,
		Months:        months,
		PricePerMonth: perMonth,
		Total:         total,
	}
}

// CalcTariffChangeOptions при снижении тарифа предлагает варианты
// для уже предоплативших арендаторов.
func CalcTariffChangeOptions(paid int64, oldPerMonth, newPerMonth int64, remainingMonths float64) []models.TariffChangeOption {
	if newPerMonth >= oldPerMonth || remainingMonths <= 0 {
		return nil
	}

	diffPerMonth := oldPerMonth - newPerMonth
	refund := int64(math.Round(float64(diffPerMonth) * remainingMonths))

	extraMonths := 0.0
	if newPerMonth > 0 {
		extraMonths = math.Round((float64(refund)/float64(newPerMonth))*100) / 100
	}
	_ = paid

	return []models.TariffChangeOption{
		{
			Action:       models.TariffActionRefund,
			Label:        "Вернуть разницу",
			RefundAmount: refund,
		},
		{
			Action:      models.TariffActionKeep,
			Label:       "Оставить оплату — получить доп. месяцы",
			ExtraMonths: extraMonths,
		},
	}
}
