import { NotificationKind } from '../models/notification';

export type NotificationTone = 'success' | 'warning' | 'danger' | 'neutral';

export function notificationTone(kind: NotificationKind): NotificationTone {
  switch (kind) {
    case 'payment_success':
      return 'success';
    case 'rental_overdue':
    case 'debt_warning':
    case 'rental_cancelled':
      return 'danger';
    case 'rental_expiring':
    case 'tariff_recalc':
    case 'request_new':
    case 'refund_pending':
    case 'resident_new':
      return 'warning';
    default:
      return 'neutral';
  }
}

export function notificationIcon(kind: NotificationKind): string {
  switch (kind) {
    case 'payment_success':
      return 'fa-wallet';
    case 'rental_expiring':
      return 'fa-hourglass-half';
    case 'rental_overdue':
      return 'fa-triangle-exclamation';
    case 'debt_warning':
      return 'fa-circle-exclamation';
    case 'tariff_changed':
    case 'tariff_recalc':
      return 'fa-percent';
    case 'request_new':
      return 'fa-inbox';
    case 'request_resolved':
      return 'fa-circle-check';
    case 'refund_pending':
      return 'fa-rotate-left';
    case 'resident_new':
      return 'fa-user-clock';
    case 'account_status':
      return 'fa-user-check';
    case 'rental_cancelled':
      return 'fa-ban';
  }
}

export const toneClass: Record<NotificationTone, string> = {
  success: 'bg-brand-accent/10 text-brand-accent',
  warning: 'bg-orange-500/10 text-orange-500',
  danger: 'bg-red-500/10 text-red-500',
  neutral: 'bg-brand-elevated text-brand-muted',
};
