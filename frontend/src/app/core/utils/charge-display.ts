import { ChargeStatus } from '../models/charge';

/** Labels for the server-computed ledger (backend repository/charges.go). */
export function chargeStatusLabel(c: { status: ChargeStatus; overdue_days: number }): string {
  switch (c.status) {
    case 'paid':
      return 'Оплачено';
    case 'overdue':
      return `Просрочка ${c.overdue_days} дн.`;
    case 'ended':
      return 'Завершена';
  }
}

/** Debt exists only for a paid (active) rental whose period has ended — never for unpaid bookings. */
export function isOverdue(r: { status: string; ends_at: string }, now = Date.now()): boolean {
  return r.status === 'active' && new Date(r.ends_at).getTime() < now;
}
