import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { environment } from '../../../environments/environment';
import {
  AccountForm,
  AddAdminResponse,
  CreateJKPayload,
  CreateJKResponse,
  JKForm,
  PlatformAdmin,
  PlatformJK,
} from '../models/platform';

/** Super-admin panel API (/platform/..., platform operator only). */
@Injectable({ providedIn: 'root' })
export class PlatformService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/platform`;

  listJKs() {
    return this.http.get<PlatformJK[]>(`${this.baseUrl}/jks`);
  }

  getJK(id: string) {
    return this.http.get<PlatformJK>(`${this.baseUrl}/jks/${id}`);
  }

  /** Creates the ЖК and its first ОСИ account in one go. */
  createJK(payload: CreateJKPayload) {
    return this.http.post<CreateJKResponse>(`${this.baseUrl}/jks`, payload);
  }

  updateJK(id: string, form: JKForm) {
    return this.http.put<PlatformJK>(`${this.baseUrl}/jks/${id}`, form);
  }

  listAdmins(jkId: string) {
    return this.http.get<PlatformAdmin[]>(`${this.baseUrl}/jks/${jkId}/admins`);
  }

  addAdmin(jkId: string, form: AccountForm) {
    return this.http.post<AddAdminResponse>(`${this.baseUrl}/jks/${jkId}/admins`, form);
  }

  resetAdminPassword(adminId: string) {
    return this.http.post<{ temporary_password: string }>(
      `${this.baseUrl}/admins/${adminId}/reset-password`,
      {},
    );
  }
}
