import { describe, expect, it } from 'vitest';

import { notificationIcon, notificationTone } from './notification-display';

describe('notification display', () => {
  it('uses a warning tone for overdue and debt', () => {
    expect(notificationTone('rental_overdue')).toBe('danger');
    expect(notificationTone('debt_warning')).toBe('danger');
    expect(notificationTone('rental_expiring')).toBe('warning');
    expect(notificationTone('tariff_recalc')).toBe('warning');
    expect(notificationTone('payment_success')).toBe('success');
    expect(notificationTone('request_resolved')).toBe('neutral');
  });

  it('has an icon for every kind', () => {
    for (const k of [
      'payment_success', 'rental_expiring', 'rental_overdue', 'debt_warning',
      'tariff_changed', 'tariff_recalc', 'request_new', 'request_resolved',
    ] as const) {
      expect(notificationIcon(k)).toMatch(/^fa-/);
    }
  });
});
