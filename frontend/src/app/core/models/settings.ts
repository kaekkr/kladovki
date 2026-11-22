export interface JKSettings {
  id: string;
  name: string;
  address?: string;
  cadastral_number?: string;
}

export interface TariffSettings {
  jk_id: string;
  amount: number; // ₸ per m² per month
}

export interface RentalSettings {
  lock_duration_minutes: number;
}

export interface AdminSettings {
  jk: JKSettings;
  tariff: TariffSettings;
  rental: RentalSettings;
}
