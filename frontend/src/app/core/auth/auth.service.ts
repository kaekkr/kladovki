import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { tap, catchError, of, finalize } from 'rxjs';
import { environment } from '../../../environments/environment';
import { LoginPayload, RegisterPayload, User } from './auth.models';
import { tokenStore, usesTokenAuth } from './token-session';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  currentUser = signal<User | null>(null);

  /** Mobile app (API on another origin): ask for a bearer token, see token-session.ts. */
  private tokenMode = usesTokenAuth(this.baseUrl, window.location.origin);
  private sessionHeaders = this.tokenMode ? new HttpHeaders({ 'X-Kladovki-Client': 'native' }) : undefined;

  /** Keeps the token (mobile app only) and stores the user without it. */
  private startSession = (user: User & { access_token?: string }) => {
    if (this.tokenMode && user.access_token) tokenStore.set(user.access_token);
    const { access_token: _token, ...rest } = user;
    this.currentUser.set(rest);
  };

  getMe() {
    return this.http.get<User>(`${this.baseUrl}/auth/me`).pipe(
      tap((user) => this.currentUser.set(user)),
      catchError(() => {
        this.currentUser.set(null);
        tokenStore.clear();
        return of(null);
      }),
    );
  }

  login(credentials: LoginPayload) {
    return this.http
      .post<User>(`${this.baseUrl}/auth/login`, credentials, { headers: this.sessionHeaders })
      .pipe(tap(this.startSession));
  }

  /** Clears the session cookie on the server; the local user is cleared even if the call fails. */
  logout() {
    return this.http.post(`${this.baseUrl}/auth/logout`, {}).pipe(
      catchError(() => of(null)),
      finalize(() => {
        this.currentUser.set(null);
        tokenStore.clear();
      }),
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
      .post<User>(`${this.baseUrl}/auth/register`, payload, { headers: this.sessionHeaders })
      .pipe(tap(this.startSession));
  }
}
