import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { environment } from '../../../environments/environment';
import { User, UserStatus, Verification } from '../auth/auth.models';

export type ReviewAction = 'approve' | 'reject' | 'block';

/** ОСИ approval of residents (only property owners may rent). */
@Injectable({ providedIn: 'root' })
export class ResidentService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  listByJK(jkId: string, status: UserStatus | '' = '') {
    return this.http.get<User[]>(`${this.baseUrl}/residents/jk/${jkId}`, { params: { status } });
  }

  pendingCount() {
    return this.http.get<{ pending: number }>(`${this.baseUrl}/residents/pending-count`);
  }

  review(userId: string, action: ReviewAction, comment = '') {
    return this.http.patch<{ message: string }>(`${this.baseUrl}/residents/${userId}/${action}`, { comment });
  }

  /** A pending/rejected user corrects their property data and asks the ОСИ again. */
  resubmit(v: Verification) {
    return this.http.put<{ message: string }>(`${this.baseUrl}/me/verification`, v);
  }
}
