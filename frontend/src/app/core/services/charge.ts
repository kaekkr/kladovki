import { Injectable, inject } from '@angular/core';
import { map } from 'rxjs/operators';
import { RentalService } from './rental';
import { Rental } from '../models/rental';
import { Charge, ChargeStatus } from '../models/charge';

@Injectable({ providedIn: 'root' })
export class ChargeService {
  private rentalService = inject(RentalService);

  listByJK(jkId: string) {
    return this.rentalService.listByJK(jkId).pipe(map((rentals) => rentals.map(toCharge)));
  }

  listDebtsByJK(jkId: string) {
    return this.listByJK(jkId).pipe(
      map(
        (charges) => charges.filter((c) => c.balance > 0).sort((a, b) => b.balance - a.balance), // largest debt first
      ),
    );
  }
}

function toCharge(r: Rental): Charge {
  const accrued = r.price_per_month * r.months;
  const paid = r.total_paid;
  const balance = accrued - paid;

  return {
    id: r.id,
    storage_number: r.storage_number,
    user_name: r.user_name,
    user_phone: r.user_phone,
    months: r.months,
    price_per_month: r.price_per_month,
    accrued,
    paid,
    balance,
    starts_at: r.starts_at,
    ends_at: r.ends_at,
    rental_status: r.status,
    status: deriveStatus(r, balance),
  };
}

function deriveStatus(r: Rental, balance: number): ChargeStatus {
  if (balance <= 0) return 'paid';
  if (r.status === 'locked') return 'pending';
  if (r.status === 'expired' || r.status === 'cancelled') {
    return balance > 0 ? 'overdue' : 'paid';
  }
  // active
  if (paidIsPartial(r)) return 'partial';
  return balance > 0 ? 'pending' : 'paid';
}

function paidIsPartial(r: Rental): boolean {
  return r.total_paid > 0 && r.total_paid < r.price_per_month * r.months;
}
