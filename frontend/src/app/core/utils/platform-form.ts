import { AccountForm, JKForm } from '../models/platform';

/**
 * Client-side checks for the super-admin forms. They mirror the server
 * (internal/service/platform_validate.go) only to give instant feedback;
 * the server validates again and is the source of truth.
 */

const COMPLEX_CADASTRAL = /^\d+(?::\d+){3,}$/;
const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Same as cadastral.Normalize on the server: no spaces, no trailing ":", upper case. */
export function normalizeCadastral(raw: string): string {
  return raw.replace(/\s+/g, '').replace(/:+$/, '').toUpperCase();
}

export function validateJKForm(form: JKForm): string | null {
  const cadastral = normalizeCadastral(form.cadastral_number);
  if (!form.name.trim()) return 'Укажите название ЖК.';
  if (!form.address.trim()) return 'Укажите адрес ЖК.';
  if (cadastral && !COMPLEX_CADASTRAL.test(cadastral)) {
    return 'Кадастровый номер ЖК должен быть вида 21:319:021:848:8.';
  }
  return null;
}

export function validateAccountForm(form: AccountForm): string | null {
  if (!form.full_name.trim()) return 'Укажите ФИО администратора ОСИ.';
  if (!EMAIL.test(form.email.trim())) return 'Укажите корректный email.';
  if (!form.phone.trim()) return 'Укажите номер телефона.';
  return null;
}

interface HttpErrorLike {
  status?: number;
  error?: unknown;
}

/** The Russian message to show for a failed request. */
export function apiErrorText(err: HttpErrorLike | null | undefined, fallback: string): string {
  const body = err?.error as { error?: unknown } | null | undefined;
  if (body && typeof body === 'object' && typeof body.error === 'string' && body.error) {
    return body.error;
  }
  if (err?.status === 0) return 'Нет связи с сервером. Проверьте интернет и попробуйте ещё раз.';
  return fallback;
}
