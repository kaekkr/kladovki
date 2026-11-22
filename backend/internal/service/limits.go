package service

import (
	"math"
	"strings"
)

// Sane bounds for money and storage parameters (QA 2026-11-22: huge values
// gave 500s, saved half-way, or overflowed price × months). The admin storage
// form in the frontend uses the same limits.
const (
	MinTariff = 1         // ₸ per м² per month
	MaxTariff = 1_000_000 // ₸ per м² per month

	MinStorageArea     = 0.01 // м², after rounding to 2 decimals (the DB keeps NUMERIC(8,2))
	MaxStorageArea     = 10_000
	MinStorageFloor    = -10
	MaxStorageFloor    = 200
	MinStorageEntrance = 0 // 0 = no entrance number (older clients omit it)
	MaxStorageEntrance = 100
	MaxStorageNumber   = 50 // characters (VARCHAR(50))
)

// ValidTariff reports whether amount is an allowed tariff.
func ValidTariff(amount int64) bool {
	return amount >= MinTariff && amount <= MaxTariff
}

// normalizeStorage trims the number, rounds the area to what the DB stores and
// checks every field against the bounds above.
func normalizeStorage(number string, area float64, floor, entrance int) (string, float64, error) {
	number = strings.TrimSpace(number)
	if number == "" || len([]rune(number)) > MaxStorageNumber {
		return "", 0, ErrInvalid
	}
	if math.IsNaN(area) || math.IsInf(area, 0) || area > MaxStorageArea+1 {
		return "", 0, ErrInvalid
	}
	area = math.Round(area*100) / 100
	if area < MinStorageArea || area > MaxStorageArea {
		return "", 0, ErrInvalid
	}
	if floor < MinStorageFloor || floor > MaxStorageFloor ||
		entrance < MinStorageEntrance || entrance > MaxStorageEntrance {
		return "", 0, ErrInvalid
	}
	return number, area, nil
}

// rentalTotal is price × months, refusing a price below 1 ₸ or an overflow.
func rentalTotal(price int64, months int) (int64, error) {
	if price < 1 || months < 1 || price > math.MaxInt64/int64(months) {
		return 0, ErrInvalid
	}
	return price * int64(months), nil
}
