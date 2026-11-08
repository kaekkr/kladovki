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

  releaseOverdue(id: string) {
    return this.http.patch<Rental>(
      `${this.baseUrl}/rentals/${id}/release-overdue`,
      {},
    );
  }

  listMine() {
    return this.http.get<Rental[]>(`${this.baseUrl}/rentals/my`);
  }

  /** Amount is calculated on the server from the rental; phone defaults to the user's phone. */
  createPayment(data: {
    rental_id: string;
    phone?: string;
    accepted_terms_version: string;
  }): Observable<{ amount: number }> {
    return this.http.post<{ amount: number }>(`${this.baseUrl}/payments`, data);
  }

  /** Sends a Kaspi invoice; once paid, the rental is extended by `months` from its current end. */
  renew(
    id: string,
    months: number,
    acceptedTermsVersion: string,
    phone?: string,
  ): Observable<{ id: string; amount: number }> {
    return this.http.post<{ id: string; amount: number }>(`${this.baseUrl}/rentals/${id}/renew`, {
      months,
      phone,
      accepted_terms_version: acceptedTermsVersion,
    });
  }
}
