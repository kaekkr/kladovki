import { Injectable, inject } from '@angular/core';
import { map } from 'rxjs/operators';
import { RentalService } from './rental';
import { Rental } from '../models/rental';
import { AdminDocument } from '../models/document';

@Injectable({ providedIn: 'root' })
export class DocumentService {
  private rentalService = inject(RentalService);

  listByJK(jkId: string) {
    return this.rentalService.listByJK(jkId).pipe(map((rentals) => rentals.flatMap(toDocuments)));
  }
}

function toDocuments(r: Rental): AdminDocument[] {
  const docs: AdminDocument[] = [];

  // Договор — almost always exists once a rental was created
  docs.push({
    id: `${r.id}-contract`,
    rental_id: r.id,
    type: 'contract',
    title: `Договор аренды №${r.storage_number}`,
    storage_number: r.storage_number,
    user_name: r.user_name,
    user_phone: r.user_phone,
    amount: null,
    created_at: r.created_at,
    period_start: r.starts_at,
    period_end: r.ends_at,
  });

  // Квитанция — only if something was paid
  if (r.total_paid > 0) {
    docs.push({
      id: `${r.id}-receipt`,
      rental_id: r.id,
      type: 'receipt',
      title: `Квитанция об оплате — кладовка №${r.storage_number}`,
      storage_number: r.storage_number,
      user_name: r.user_name,
      user_phone: r.user_phone,
      amount: r.total_paid,
      created_at: r.starts_at, // approximate; later use payment.created_at
      period_start: r.starts_at,
      period_end: r.ends_at,
    });
  }

  // Акт — for active rentals
  if (r.status === 'active') {
    docs.push({
      id: `${r.id}-act`,
      rental_id: r.id,
      type: 'act',
      title: `Акт приёма-передачи — кладовка №${r.storage_number}`,
      storage_number: r.storage_number,
      user_name: r.user_name,
      user_phone: r.user_phone,
      amount: null,
      created_at: r.starts_at,
      period_start: r.starts_at,
      period_end: r.ends_at,
    });
  }

  return docs;
}
