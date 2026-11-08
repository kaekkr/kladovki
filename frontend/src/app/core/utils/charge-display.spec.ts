import { describe, expect, it } from 'vitest';

import { chargeStatusLabel, isOverdue } from './charge-display';

describe('charge display', () => {
  it('labels server charge statuses', () => {
    expect(chargeStatusLabel({ status: 'paid', overdue_days: 0 })).toBe('Оплачено');
    expect(chargeStatusLabel({ status: 'overdue', overdue_days: 31 })).toBe('Просрочка 31 дн.');
    expect(chargeStatusLabel({ status: 'ended', overdue_days: 0 })).toBe('Завершена');
  });

  it('a paid rental is overdue once its paid period has ended', () => {
    const now = new Date('2026-11-09T10:00:00Z').getTime();
    expect(isOverdue({ status: 'active', ends_at: '2026-11-08T10:00:00Z' }, now)).toBe(true);
    expect(isOverdue({ status: 'active', ends_at: '2026-12-08T10:00:00Z' }, now)).toBe(false);
    // An unpaid booking whose hold passed is not a debt.
    expect(isOverdue({ status: 'locked', ends_at: '2026-11-08T10:00:00Z' }, now)).toBe(false);
    expect(isOverdue({ status: 'expired', ends_at: '2026-11-08T10:00:00Z' }, now)).toBe(false);
  });
});
