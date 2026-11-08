export type NotificationKind =
  | 'payment_success'
  | 'rental_expiring'
  | 'rental_overdue'
  | 'debt_warning'
  | 'tariff_changed'
  | 'tariff_recalc'
  | 'request_new'
  | 'request_resolved'
  | 'refund_pending'
  | 'resident_new'
  | 'account_status'
  | 'rental_cancelled'
  | 'transfer_new';

export interface AppNotification {
  id: string;
  kind: NotificationKind;
  title: string;
  body: string;
  link: string;
  read: boolean;
  created_at: string;
}

export interface NotificationChannels {
  email: boolean;
  telegram: boolean;
  sms: boolean;
  whatsapp: boolean;
  telegram_available: boolean;
  email_address: string;
}
