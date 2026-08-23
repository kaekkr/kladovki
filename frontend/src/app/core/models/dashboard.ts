export type DashboardPeriod = '7d' | '30d' | 'quarter';

export interface DashboardStats {
  received: number;
  receivedChange: number;

  debt: number;
  debtApartments: number;

  occupiedStorages: number;
  totalStorages: number;

  activeRentals: number;
  newRentals: number;
}

export interface RevenuePoint {
  date: string;
  amount: number;
}

export interface DashboardRevenue {
  points: RevenuePoint[];
}

export type AttentionType = 'rental_expiring' | 'storage_locked';

export interface DashboardAttention {
  type: AttentionType;

  rental_id?: string;
  storage_id: string;

  storage_number: string;

  user_id: string;
  full_name: string;
  phone: string;

  ends_at: string;

  days_left?: number;
}

export interface DashboardActivity {
  id: string;
  type: 'payment' | 'rental' | 'expired';
  user_id: string;
  full_name: string;
  storage_number: string;
  amount?: number;
  rental_id?: string;
  created_at: string;
}

export interface DashboardOccupancy {
  total: number;
  free: number;
  occupied: number;
  locked: number;
}
