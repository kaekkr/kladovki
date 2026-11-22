import { formatDate } from '@angular/common';

/** Same as the date pipe defaults in app.config: Kazakhstan time. */
export const KZ_TIMEZONE = '+0500';

/** One date format on resident screens: «24 марта 2027» (same as `| date: 'd MMMM y'`). */
export function formatRuDate(value: string | Date | null | undefined): string {
  if (!value) return '—';
  const d = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return formatDate(d, 'd MMMM y', 'ru', KZ_TIMEZONE);
}

/** «24 марта» — for compact places where the year is obvious. */
export function formatRuDayMonth(value: string | Date): string {
  return formatDate(value, 'd MMMM', 'ru', KZ_TIMEZONE);
}

/** «14:05» in Kazakhstan time. */
export function formatRuTime(value: string | Date): string {
  return formatDate(value, 'HH:mm', 'ru', KZ_TIMEZONE);
}

/** Residents pay through Kaspi; the gateway name (ApiPay) means nothing to them. */
export function paymentProviderLabel(provider: string): string {
  switch ((provider ?? '').toLowerCase()) {
    case 'kaspi':
    case 'apipay':
    case '':
      return 'Kaspi';
    case 'card':
      return 'Банковская карта';
    default:
      return provider;
  }
}

/** «7 500 ₸». */
export function formatTenge(value: number): string {
  return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 0 }).format(value) + ' ₸';
}
