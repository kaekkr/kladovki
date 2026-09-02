import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { environment } from '../../../environments/environment.development';
import { Payment } from '../models/payment';

@Injectable({
  providedIn: 'root',
})
export class PaymentService {
  private http = inject(HttpClient);

  private baseUrl = environment.apiUrl;

  listByJK(jkId: string) {
    return this.http.get<Payment[]>(`${this.baseUrl}/payments/jk/${jkId}`);
  }

  getById(id: string) {
    return this.http.get<Payment>(`${this.baseUrl}/payments/${id}`);
  }
}
