import { Rental } from '../models/rental';
import { TenantRequest } from '../models/request';

const DAY_MS = 86_400_000;

type DebtFields = Pick<Rental, 'status' | 'ends_at' | 'price_per_month'>;

/**
 * Started days since the paid period ended — same rule as the server's charges
 * ledger (backend/internal/repository/charges.go): CEIL(seconds / 86400).
 */
export function overdueDays(rental: DebtFields, now: Date = new Date()): number {
  if (rental.status !== 'active') return 0;
  const late = now.getTime() - new Date(rental.ends_at).getTime();
  return late > 0 ? Math.ceil(late / DAY_MS) : 0;
}

/**
 * Approximate debt of an overdue rental: every started day costs price/30,
 * rounded like the server does. The resident has no charges endpoint, so the
 * app computes it the same way; the ОСИ's ledger is the authority.
 */
export function rentalDebt(rental: DebtFields, now: Date = new Date()): number {
  return Math.round((rental.price_per_month * overdueDays(rental, now)) / 30);
}

/**
 * New end date if the tenant chooses «extend» after a tariff decrease:
 * the remaining time is stretched to the new price,
 * `ends_at + (ends_at − now) × (old/new − 1)` (docs/domain.md, «Tariff change»).
 */
export function tariffExtendEndsAt(
  endsAt: string,
  oldPrice: number,
  newPrice: number,
  now: Date = new Date(),
): string {
  const end = new Date(endsAt).getTime();
  const remaining = end - now.getTime();
  if (remaining <= 0 || newPrice <= 0 || oldPrice <= newPrice) {
    return new Date(end).toISOString();
  }
  return new Date(end + remaining * (oldPrice / newPrice - 1)).toISOString();
}

type RefundFields = Pick<TenantRequest, 'rental_id' | 'type' | 'status' | 'choice' | 'amount'>;

/**
 * Money the ОСИ still has to return for a tariff decrease the tenant chose to
 * refund (fallback when the rental has no `refund_pending` field). `total_paid`
 * is reduced by the server only when the ОСИ confirms the money was returned.
 */
export function pendingTariffRefund(rentalId: string, requests: readonly RefundFields[]): number {
  return requests
    .filter(
      (r) =>
        r.rental_id === rentalId &&
        r.type === 'tariff_recalc' &&
        r.choice === 'refund' &&
        r.status === 'open',
    )
    .reduce((sum, r) => sum + r.amount, 0);
}

function addUtcMonths(date: Date, months: number): Date {
  const d = new Date(date.getTime());
  d.setUTCMonth(d.getUTCMonth() + months);
  return d;
}

/**
 * «12 мес.», «7 мес. 12 дн.», «20 дн.» — the actual paid period from the dates,
 * so it stays true after a renewal or a tariff extension (months × price may not).
 */
export function rentalPeriodLabel(startsAt: string, endsAt: string): string {
  const start = new Date(startsAt);
  const end = new Date(endsAt);
  if (!(end.getTime() > start.getTime())) return '—';
  let months =
    (end.getUTCFullYear() - start.getUTCFullYear()) * 12 +
    (end.getUTCMonth() - start.getUTCMonth());
  while (months > 0 && addUtcMonths(start, months).getTime() > end.getTime()) months--;
  const days = Math.round((end.getTime() - addUtcMonths(start, months).getTime()) / DAY_MS);
  if (months === 0) return `${Math.max(days, 1)} дн.`;
  return days > 0 ? `${months} мес. ${days} дн.` : `${months} мес.`;
}
