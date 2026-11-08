import { TenantRequest } from '../models/request';

export function requestTitle(r: TenantRequest): string {
  switch (r.type) {
    case 'tariff_recalc':
      return 'Перерасчёт после снижения тарифа';
    case 'early_termination':
      return 'Досрочное завершение аренды';
    case 'payment_refund':
      return 'Возврат оплаты';
  }
}

export function requestStatusLabel(r: TenantRequest): string {
  switch (r.status) {
    case 'awaiting_choice':
      return 'Ожидает выбора арендатора';
    case 'open':
      return 'На рассмотрении ОСИ';
    case 'rejected':
      return 'Отклонено';
    case 'done':
      if (r.type === 'tariff_recalc') {
        return r.choice === 'extend' ? 'Период продлён' : 'Возврат выполнен';
      }
      return r.type === 'payment_refund' ? 'Возврат выполнен' : 'Одобрено';
  }
}

export function needsOsiAction(r: TenantRequest): boolean {
  return r.status === 'open';
}
