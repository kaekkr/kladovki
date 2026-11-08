import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

import { environment } from '../../../environments/environment';
import { TenantRequest } from '../models/request';

@Injectable({ providedIn: 'root' })
export class RequestService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  listMine() {
    return this.http.get<TenantRequest[]>(`${this.baseUrl}/requests/my`);
  }

  listByJK(jkId: string) {
    return this.http.get<TenantRequest[]>(`${this.baseUrl}/requests/jk/${jkId}`);
  }

  openCount() {
    return this.http.get<{ open: number }>(`${this.baseUrl}/requests/open-count`);
  }

  choose(id: string, choice: 'extend' | 'refund') {
    return this.http.post<{ message: string }>(`${this.baseUrl}/requests/${id}/choice`, { choice });
  }

  terminateRental(rentalId: string, comment: string) {
    return this.http.post<TenantRequest>(`${this.baseUrl}/rentals/${rentalId}/terminate`, {
      comment,
    });
  }

  resolve(id: string, action: 'approve' | 'reject', comment: string) {
    return this.http.patch<{ message: string }>(`${this.baseUrl}/requests/${id}/resolve`, {
      action,
      comment,
    });
  }
}
