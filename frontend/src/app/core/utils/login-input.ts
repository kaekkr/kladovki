export type LoginKind = 'iin' | 'email' | 'phone' | 'invalid';

/** Residents log in with IIN (12 digits); OSI admins with email; phone kept for old accounts. */
export function classifyLogin(raw: string): LoginKind {
  const v = raw.trim();
  if (/^\d{12}$/.test(v)) return 'iin';
  if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return 'email';
  const digits = v.replace(/\D/g, '');
  if (digits.length === 11 && /^[78]/.test(digits)) return 'phone';
  return 'invalid';
}

export function passwordsProblem(password: string, repeat: string): string | null {
  if (password.length < 6) return 'Пароль должен быть не короче 6 символов';
  if (password !== repeat) return 'Пароли не совпадают';
  return null;
}
