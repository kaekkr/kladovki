export type PaymentStatus = 'pending' | 'success' | 'failed';

export interface Payment {
  id: string;
  rental_id: string;
  user_id: string;

  // From DTO Join
  user_name: string;
  storage_number: string;

  amount: number;
  provider: string; // 'kaspi', 'card', etc.
  status: PaymentStatus;

  created_at: string;
}
