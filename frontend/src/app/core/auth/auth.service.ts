import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap, catchError, of, finalize } from 'rxjs';
import { environment } from '../../../environments/environment';
import { LoginPayload, RegisterPayload, User } from './auth.models';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  currentUser = signal<User | null>(null);

  getMe() {
    return this.http.get<User>(`${this.baseUrl}/auth/me`).pipe(
      tap((user) => this.currentUser.set(user)),
      catchError(() => {
        this.currentUser.set(null);
        return of(null);
      }),
    );
  }

  login(credentials: LoginPayload) {
    return this.http
      .post<User>(`${this.baseUrl}/auth/login`, credentials)
      .pipe(tap((user) => this.currentUser.set(user)));
  }

  /** Clears the session cookie on the server; the local user is cleared even if the call fails. */
  logout() {
    return this.http.post(`${this.baseUrl}/auth/logout`, {}).pipe(
      catchError(() => of(null)),
      finalize(() => this.currentUser.set(null)),
    );
  }

  changePassword(currentPassword: string, newPassword: string) {
    return this.http.put<{ message: string }>(`${this.baseUrl}/me/password`, {
      current_password: currentPassword,
      new_password: newPassword,
    });
  }

  /** Whether an IIN already has an account (one password field) or needs first-login setup. */
  iinStatus(iin: string) {
    return this.http.post<{ registered: boolean }>(`${this.baseUrl}/auth/iin-status`, { iin });
  }

  register(payload: RegisterPayload) {
    return this.http
      .post<User>(`${this.baseUrl}/auth/register`, payload)
      .pipe(tap((user) => this.currentUser.set(user)));
  }
}
