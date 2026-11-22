package repository

// Phones stored before normalization (2026-11) may read 87001234567 or
// "8 (700) 123-45-67"; new ones are +77001234567. They are the same number,
// so uniqueness and phone login compare a digit key instead of the text.

// phoneKeySQL is the digit key of a phone expression: digits only, a leading
// 8 of an 11-digit number read as 7 (+7 700 … and 8 700 … → 7700…).
func phoneKeySQL(expr string) string {
	digits := `regexp_replace(` + expr + `, '\D', '', 'g')`
	return `(CASE WHEN ` + digits + ` ~ '^8[0-9]{10}$' THEN '7' || substr(` + digits + `, 2) ELSE ` + digits + ` END)`
}

// samePhoneSQL: column col holds the same number as the parameter param.
// The superadmin placeholder (superadmin:…) only ever matches itself exactly.
func samePhoneSQL(col, param string) string {
	return `(` + col + ` = ` + param + ` OR (` + col + ` NOT LIKE 'superadmin:%' AND ` +
		phoneKeySQL(param) + ` <> '' AND ` + phoneKeySQL(col) + ` = ` + phoneKeySQL(param) + `))`
}
