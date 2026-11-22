/**
 * Kazakh mobile numbers. People type them every which way: «+7 777 123 45 67»,
 * «8 (777) 123-45-67», «87771234567». The server stores one canonical form,
 * `+77XXXXXXXXX` (+7 and 10 digits, the first of them 7), and enforces the same rule.
 */
export const PHONE_PROBLEM = 'Введите мобильный номер Казахстана, например +7 777 123 45 67';

/** Canonical `+77XXXXXXXXX`, or null when this is not a Kazakh mobile number. */
export function normalizePhone(raw: string): string | null {
  const digits = (raw ?? '').replace(/\D/g, '');
  const national =
    digits.length === 11 && (digits.startsWith('7') || digits.startsWith('8'))
      ? digits.slice(1)
      : digits.length === 10
        ? digits
        : null;
  if (!national || !national.startsWith('7')) return null;
  return `+7${national}`;
}

/** A Russian message for the form, or null when the number is fine. */
export function phoneProblem(raw: string): string | null {
  if (!(raw ?? '').trim()) return 'Укажите номер телефона';
  return normalizePhone(raw) ? null : PHONE_PROBLEM;
}

/** «+7 777 123 45 67» for display; anything unparseable is shown as is. */
export function formatPhone(raw: string): string {
  const p = normalizePhone(raw);
  if (!p) return raw ?? '';
  return `+7 ${p.slice(2, 5)} ${p.slice(5, 8)} ${p.slice(8, 10)} ${p.slice(10)}`;
}
