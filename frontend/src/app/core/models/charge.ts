export type ChargeStatus = 'paid' | 'overdue' | 'ended';

/** One paid rental in the ЖК ledger. Debt is computed on the server (GET /charges/jk/:id). */
export interface Charge {
  id: string; // rental id
  storage_number: string;
  user_name: string;
  user_phone: string;
  company_name?: string;
  months: number;
  price_per_month: number;
  accrued: number; // paid + debt
  paid: number;
  balance: number; // debt: started overdue days × monthly price / 30
  overdue_days: number;
  starts_at: string;
  ends_at: string; // paid until
  status: ChargeStatus;
}

/** Shape returned by the backend. */
export interface ChargeRow extends Omit<Charge, 'id' | 'balance'> {
  rental_id: string;
  debt: number;
}
