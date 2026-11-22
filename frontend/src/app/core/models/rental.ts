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
  /** Money the ОСИ still has to return (tariff refund chosen, not yet paid back). total_paid is not reduced until then. */
  refund_pending?: number;

  starts_at: string;
  ends_at: string;

  status: RentalStatus;
  created_at: string;

  /** Record of a forced termination for debt (agreement п. 5.2), if any. */
  termination?: Termination;
}

export interface Termination {
  reason: string;
  note: string;
  police: boolean;
  costs: number;
  created_at: string;
}
