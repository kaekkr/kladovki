import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

// import { environment } from '../../../environments/environment';
import { environment } from '../../../environments/environment';
import { Payment } from '../models/payment';

@Injectable({
  providedIn: 'root',
})
export class PaymentService {
  private http = inject(HttpClient);

  private baseUrl = environment.apiUrl;

  /** Optional period: YYYY-MM-DD, inclusive Kazakhstan days. */
  listByJK(jkId: string, period?: { from: string; to: string }) {
    const params: Record<string, string> = {};
    if (period?.from) params['from'] = period.from;
    if (period?.to) params['to'] = period.to;
    return this.http.get<Payment[]>(`${this.baseUrl}/payments/jk/${jkId}`, { params });
  }

  listMine() {
    return this.http.get<Payment[]>(`${this.baseUrl}/payments/my`);
  }

  getById(id: string) {
    return this.http.get<Payment>(`${this.baseUrl}/payments/${id}`);
  }
}
