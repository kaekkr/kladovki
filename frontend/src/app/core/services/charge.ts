import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { map } from 'rxjs/operators';

import { environment } from '../../../environments/environment';
import { Charge, ChargeRow } from '../models/charge';

/** ЖК ledger (charged / paid / debt), computed on the server. */
@Injectable({ providedIn: 'root' })
export class ChargeService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  listByJK(jkId: string) {
    return this.http
      .get<ChargeRow[]>(`${this.baseUrl}/charges/jk/${jkId}`)
      .pipe(map((rows) => rows.map(({ rental_id, debt, ...rest }) => ({ ...rest, id: rental_id, balance: debt }) as Charge)));
  }

  /** Rentals that owe money, largest debt first (the server already sorts by debt). */
  listDebtsByJK(jkId: string) {
    return this.listByJK(jkId).pipe(map((charges) => charges.filter((c) => c.balance > 0)));
  }
}
