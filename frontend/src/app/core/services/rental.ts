import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

// import { environment } from '../../../environments/environment.development';
import { environment } from '../../../environments/environment';
import { Rental } from '../models/rental';

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

  cancel(id: string) {
    return this.http.patch<Rental>(`${this.baseUrl}/rentals/${id}/cancel`, {});
  }

  forceRelease(id: string) {
    return this.http.patch<Rental>(`${this.baseUrl}/rentals/${id}/force-release`, {});
  }
}
