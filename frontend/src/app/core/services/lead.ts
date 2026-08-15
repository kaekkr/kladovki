import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment.development';

export interface LeadPayload {
  full_name: string;
  phone: string;
  email?: string;
}

@Injectable({ providedIn: 'root' })
export class LeadService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  submit(data: LeadPayload) {
    return this.http.post(`${this.baseUrl}/lead`, data);
  }
}
