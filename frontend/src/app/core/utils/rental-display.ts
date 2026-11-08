import { Rental } from '../models/rental';

/** A booking that expired before payment arrived — it never became a rental. */
export function isUnpaidExpired(rental: Rental): boolean {
  return rental.status === 'expired' && rental.total_paid === 0;
}

/** Pending booking: price of the chosen period (amount due). Otherwise: what was actually paid. */
export function rentalTotalCost(rental: Rental): number {
  return rental.status === 'locked'
    ? rental.price_per_month * rental.months
    : rental.total_paid;
}

/**
 * While booked, `ends_at` is the 2-minute lock deadline, not the rental end.
 * Show start + chosen months instead (the backend sets the real end on payment).
 */
export function rentalDisplayEndsAt(rental: Rental): string {
  if (rental.status !== 'locked' && !isUnpaidExpired(rental)) {
    return rental.ends_at;
  }
  const end = new Date(rental.starts_at);
  end.setUTCMonth(end.getUTCMonth() + rental.months);
  return end.toISOString();
}

export function rentalStatusLabel(rental: Rental): string {
  if (isUnpaidExpired(rental)) return 'Бронь истекла';
  switch (rental.status) {
    case 'locked':
      return 'Ожидает оплаты';
    case 'active':
      return 'Активна';
    case 'expired':
      return 'Завершена';
    case 'cancelled':
      return 'Отменена';
    default:
      return rental.status;
  }
}

/** A paid rental (overdue included) can be extended by paying more months. */
export function canRenew(rental: Rental): boolean {
  return rental.status === 'active';
}

/** Renewal continues from the current end, so overdue time is covered first. */
export function renewalEndsAt(rental: Rental, months: number): string {
  const end = new Date(rental.ends_at);
  end.setUTCMonth(end.getUTCMonth() + months);
  return end.toISOString();
}

/** Same formula as the server: round(area × tariff) per month. */
export function renewalPrice(area: number, tariff: number, months: number): number {
  return Math.round(area * tariff) * months;
}
