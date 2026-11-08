import { describe, expect, it } from 'vitest';

import { TenantRequest } from '../models/request';
import { requestStatusLabel, requestTitle, needsOsiAction } from './request-display';

function req(overrides: Partial<TenantRequest> = {}): TenantRequest {
  return {
    id: 'q1', jk_id: 'jk', user_id: 'u', user_name: 'Иван', user_phone: '+7700',
    rental_id: 'r', storage_number: '12', type: 'tariff_recalc', status: 'awaiting_choice',
    amount: 1000, choice: '', old_price: 5000, new_price: 4000, comment: '', resolution: '',
    created_at: '2026-11-08T10:00:00Z', resolved_at: null, ...overrides,
  };
}

describe('requestTitle', () => {
  it('names each request type for people', () => {
    expect(requestTitle(req())).toBe('Перерасчёт после снижения тарифа');
    expect(requestTitle(req({ type: 'early_termination' }))).toBe('Досрочное завершение аренды');
    expect(requestTitle(req({ type: 'payment_refund' }))).toBe('Возврат оплаты');
  });
});

describe('requestStatusLabel', () => {
  it('describes who the request is waiting for', () => {
    expect(requestStatusLabel(req())).toBe('Ожидает выбора арендатора');
    expect(requestStatusLabel(req({ status: 'open' }))).toBe('На рассмотрении ОСИ');
    expect(requestStatusLabel(req({ status: 'done', choice: 'extend' }))).toBe('Период продлён');
    expect(requestStatusLabel(req({ status: 'done', choice: 'refund' }))).toBe('Возврат выполнен');
    expect(requestStatusLabel(req({ status: 'done', type: 'early_termination' }))).toBe('Одобрено');
    expect(requestStatusLabel(req({ status: 'rejected' }))).toBe('Отклонено');
  });
});

describe('needsOsiAction', () => {
  it('is true only for open requests', () => {
    expect(needsOsiAction(req({ status: 'open' }))).toBe(true);
    expect(needsOsiAction(req())).toBe(false);
    expect(needsOsiAction(req({ status: 'done' }))).toBe(false);
  });
});
