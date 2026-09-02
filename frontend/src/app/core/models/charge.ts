export type ChargeStatus = 'paid' | 'partial' | 'pending' | 'overdue';

export interface Charge {
  id: string; // rental id
  storage_number: string;
  user_name: string;
  user_phone: string;
  months: number;
  price_per_month: number;
  accrued: number; // price * months
  paid: number; // total_paid
  balance: number; // accrued - paid
  starts_at: string;
  ends_at: string;
  rental_status: string;
  status: ChargeStatus;
}
