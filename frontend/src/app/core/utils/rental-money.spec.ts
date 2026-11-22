import { describe, expect, it } from 'vitest';

import {
  overdueDays,
  pendingTariffRefund,
  rentalDebt,
  rentalPeriodLabel,
  tariffExtendEndsAt,
} from './rental-money';

describe('pendingTariffRefund', () => {
  const base = { rental_id: 'r1', type: 'tariff_recalc' as const, amount: 1200 };
  it('sums open refund choices of this rental only', () => {
    expect(
      pendingTariffRefund('r1', [
        { ...base, status: 'open', choice: 'refund' },
        { ...base, status: 'done', choice: 'refund' },
        { ...base, status: 'done', choice: 'extend' },
        { ...base, status: 'awaiting_choice', choice: '' },
        { ...base, rental_id: 'r2', status: 'open', choice: 'refund' },
        { ...base, type: 'early_termination', status: 'open', choice: '' },
      ]),
    ).toBe(1200);
    expect(pendingTariffRefund('r1', [])).toBe(0);
  });
});

describe('rentalPeriodLabel', () => {
  it('shows whole months and the leftover days', () => {
    expect(rentalPeriodLabel('2026-03-24T10:00:00Z', '2027-03-24T10:00:00Z')).toBe('12 мес.');
    expect(rentalPeriodLabel('2026-03-24T10:00:00Z', '2026-11-05T10:00:00Z')).toBe('7 мес. 12 дн.');
    expect(rentalPeriodLabel('2026-03-24T10:00:00Z', '2026-04-13T10:00:00Z')).toBe('20 дн.');
  });

  it('is a dash for an empty or reversed period', () => {
    expect(rentalPeriodLabel('2026-03-24T10:00:00Z', '2026-03-24T10:00:00Z')).toBe('—');
  });
});

const now = new Date('2026-11-22T12:00:00Z');

describe('overdueDays / rentalDebt', () => {
  it('counts every started day after ends_at', () => {
    const r = { status: 'active' as const, ends_at: '2026-11-12T12:00:00Z', price_per_month: 7500 };
    expect(overdueDays(r, now)).toBe(10);
    expect(rentalDebt(r, now)).toBe(2500);
  });

  it('rounds a partial day up and the money to whole tenge', () => {
    const r = { status: 'active' as const, ends_at: '2026-11-22T11:00:00Z', price_per_month: 7000 };
    expect(overdueDays(r, now)).toBe(1);
    expect(rentalDebt(r, now)).toBe(233);
  });

  it('is zero for a prepaid or ended rental', () => {
    expect(
      rentalDebt({ status: 'active', ends_at: '2027-03-24T00:00:00Z', price_per_month: 7500 }, now),
    ).toBe(0);
    expect(
      rentalDebt(
        { status: 'expired', ends_at: '2026-01-01T00:00:00Z', price_per_month: 7500 },
        now,
      ),
    ).toBe(0);
  });
});

describe('tariffExtendEndsAt', () => {
  it('stretches the remaining time by old/new − 1', () => {
    // 30 days left, price 10 000 → 5 000: 30 more days.
    expect(tariffExtendEndsAt('2026-12-22T12:00:00Z', 10000, 5000, now)).toBe(
      '2027-01-21T12:00:00.000Z',
    );
  });

  it('keeps the end date when nothing is left or the price did not drop', () => {
    expect(tariffExtendEndsAt('2026-11-01T00:00:00Z', 10000, 5000, now)).toBe(
      '2026-11-01T00:00:00.000Z',
    );
    expect(tariffExtendEndsAt('2026-12-22T12:00:00Z', 5000, 5000, now)).toBe(
      '2026-12-22T12:00:00.000Z',
    );
    expect(tariffExtendEndsAt('2026-12-22T12:00:00Z', 5000, 0, now)).toBe(
      '2026-12-22T12:00:00.000Z',
    );
  });
});
