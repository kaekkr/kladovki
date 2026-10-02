import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { environment } from '../../../environments/environment';
import { Rental } from '../models/rental';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class RentalService {
  private http = inject(HttpClient);

  private baseUrl = environment.apiUrl;

  listByJK(jkId: string) {
    return this.http.get<Rental[]>(`${this.baseUrl}/rentals/jk/${jkId}`);
  }

  getById(id: string) {
    return this.http.get<Rental>(`${this.baseUrl}/rentals/${id}`);
  }

  lockStorage(storageId: string, months: number) {
    return this.http.post<Rental>(`${this.baseUrl}/rentals/lock`, {
      storage_id: storageId,
      months,
    });
  }

  confirmPayment(rentalId: string) {
    return this.http.post<Rental>(`${this.baseUrl}/rentals/confirm-payment`, {
      rental_id: rentalId,
    });
  }

  cancel(id: string) {
    return this.http.patch<Rental>(`${this.baseUrl}/rentals/${id}/cancel`, {});
  }

  forceRelease(id: string) {
    return this.http.patch<Rental>(`${this.baseUrl}/rentals/${id}/force-release`, {});
  }

  createPayment(data: {
    rental_id: string;
    amount: number;
    phone: string;
  }): Observable<{ payment_url: string }> {
    return this.http.post<{ payment_url: string }>(`${this.baseUrl}/payments`, data);
  }
}
