export type RentalStatus = 'locked' | 'active' | 'expired' | 'cancelled';

export interface Rental {
  id: string;

  storage_id: string;
  storage_number: string;

  user_id: string;
  user_name: string;
  user_phone: string;

  jk_id: string;

  months: number;
  price_per_month: number;
  total_paid: number;

  starts_at: string;
  ends_at: string;

  status: RentalStatus;
  created_at: string;
}
