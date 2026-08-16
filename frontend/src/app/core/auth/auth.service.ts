import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { environment } from '../../../environments/environment.development';

export interface User {
  id: string;
  email: string;
  role: 'admin' | 'superadmin' | 'client';
  full_name?: string;
  jk_id?: string;
}

export interface LoginPayload {
  email: string;
  password_hash: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  // Reactive state using Angular Signals
  currentUser = signal<User | null>(null);

  login(credentials: LoginPayload) {
    return this.http.post<User>(`${this.baseUrl}/auth/login`, credentials).pipe(
      tap((user) => {
        this.currentUser.set(user);
      }),
    );
  }

  logout() {
    return this.http.post(`${this.baseUrl}/auth/logout`, {}).pipe(
      tap(() => {
        this.currentUser.set(null);
      }),
    );
  }
}
