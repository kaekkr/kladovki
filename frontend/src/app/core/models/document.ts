export type DocumentType = 'contract' | 'receipt' | 'act';

export interface AdminDocument {
  id: string; // unique key (rentalId + type)
  rental_id: string;
  type: DocumentType;
  title: string;
  storage_number: string;
  user_name: string;
  user_phone: string;
  amount: number | null; // for receipt
  created_at: string;
  period_start: string;
  period_end: string;
}
