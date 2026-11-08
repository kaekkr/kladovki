import { describe, expect, it } from 'vitest';

import { Rental } from '../models/rental';
import {
  canRenew,
  isUnpaidExpired,
  rentalDisplayEndsAt,
  rentalStatusLabel,
  rentalTotalCost,
  renewalEndsAt,
  renewalPrice,
} from './rental-display';

function rental(overrides: Partial<Rental> = {}): Rental {
  return {
    id: 'r1',
    storage_id: 's1',
    storage_number: '1',
    user_id: 'u1',
    user_name: 'Test',
    user_phone: '+77000000000',
    jk_id: 'jk1',
    months: 2,
    price_per_month: 59_400,
    total_paid: 0,
    starts_at: '2026-10-30T10:00:00.000Z',
    ends_at: '2026-10-30T10:02:00.000Z', // 2-minute lock deadline
    status: 'locked',
    created_at: '2026-10-30T10:00:00.000Z',
    ...overrides,
  };
}

describe('rentalTotalCost', () => {
  it('shows the full price of the chosen period while booked (not 0)', () => {
    expect(rentalTotalCost(rental())).toBe(118_800);
  });

  it('shows 0 for a booking that expired unpaid (nothing was charged)', () => {
    expect(rentalTotalCost(rental({ status: 'expired' }))).toBe(0);
  });

  it('shows the amount actually paid for an active rental', () => {
    expect(rentalTotalCost(rental({ status: 'active', total_paid: 118_800 }))).toBe(118_800);
  });
});

describe('rentalDisplayEndsAt', () => {
  it('shows start + chosen months while booked, not the lock deadline', () => {
    const end = new Date(rentalDisplayEndsAt(rental()));
    expect(end.getUTCFullYear()).toBe(2026);
    expect(end.getUTCMonth()).toBe(11); // December
    expect(end.getUTCDate()).toBe(30);
  });

  it('uses the stored end date for an active rental', () => {
    const r = rental({ status: 'active', total_paid: 1, ends_at: '2026-12-30T10:00:00.000Z' });
    expect(rentalDisplayEndsAt(r)).toBe('2026-12-30T10:00:00.000Z');
  });
});

describe('isUnpaidExpired / rentalStatusLabel', () => {
  it('treats an expired booking without payment as an expired booking, not a finished rental', () => {
    const r = rental({ status: 'expired' });
    expect(isUnpaidExpired(r)).toBe(true);
    expect(rentalStatusLabel(r)).toBe('Бронь истекла');
  });

  it('labels a paid expired rental as finished', () => {
    const r = rental({ status: 'expired', total_paid: 118_800 });
    expect(isUnpaidExpired(r)).toBe(false);
    expect(rentalStatusLabel(r)).toBe('Завершена');
  });

  it('labels a pending booking as awaiting payment', () => {
    expect(rentalStatusLabel(rental())).toBe('Ожидает оплаты');
  });

  it('labels active and cancelled rentals', () => {
    expect(rentalStatusLabel(rental({ status: 'active', total_paid: 1 }))).toBe('Активна');
    expect(rentalStatusLabel(rental({ status: 'cancelled' }))).toBe('Отменена');
  });
});

describe('renewal', () => {
  it('only an active (paid) rental can be renewed, overdue included', () => {
    expect(canRenew(rental({ status: 'active', total_paid: 100 }))).toBe(true);
    expect(canRenew(rental({ status: 'active', ends_at: '2020-01-01T00:00:00.000Z' }))).toBe(true);
    expect(canRenew(rental({ status: 'locked' }))).toBe(false);
    expect(canRenew(rental({ status: 'expired' }))).toBe(false);
    expect(canRenew(rental({ status: 'cancelled' }))).toBe(false);
  });

  it('extends from the current end date, so overdue time is paid first', () => {
    const r = rental({ status: 'active', ends_at: '2026-10-15T09:00:00.000Z' });
    expect(renewalEndsAt(r, 3)).toBe('2027-01-15T09:00:00.000Z');
  });

  it('prices by area × tariff per month', () => {
    expect(renewalPrice(5, 1000, 3)).toBe(15_000);
    expect(renewalPrice(2.5, 999, 1)).toBe(2498); // rounded per month like the server
  });
});
