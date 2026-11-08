export type RequestType = 'tariff_recalc' | 'early_termination' | 'payment_refund';
export type RequestStatus = 'awaiting_choice' | 'open' | 'done' | 'rejected';

export interface TenantRequest {
  id: string;
  jk_id: string;
  user_id: string;
  user_name: string;
  user_phone: string;
  rental_id: string;
  storage_number: string;
  type: RequestType;
  status: RequestStatus;
  amount: number;
  choice: '' | 'extend' | 'refund';
  old_price: number;
  new_price: number;
  comment: string;
  resolution: string;
  created_at: string;
  resolved_at: string | null;
}
