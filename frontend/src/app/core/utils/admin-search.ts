/** Digits only; a KZ trunk prefix 8 becomes 7, so 8-700…, +7 700… and 700… compare equal. */
function phoneDigits(s: string): string {
  return s.replace(/\D/g, '').replace(/^8(?=\d{3})/, '7');
}

/** Search box rule for ОСИ lists: text contains the query, or the digits do (phones, ИИН). */
export function matchesSearch(fields: (string | null | undefined)[], rawQuery: string): boolean {
  const q = rawQuery.trim().toLowerCase();
  if (!q) return true;
  const qDigits = phoneDigits(q);
  const digitQuery = qDigits.length >= 3 && /^[\d\s()+-]+$/.test(q);
  return fields.some((f) => {
    if (!f) return false;
    if (f.toLowerCase().includes(q)) return true;
    return digitQuery && phoneDigits(f).includes(qDigits);
  });
}
