import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { environment } from '../../../environments/environment.development';
import { LoginPayload, User } from './auth.models';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

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
