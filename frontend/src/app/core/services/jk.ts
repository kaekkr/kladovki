import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap, catchError, of } from 'rxjs';
import { environment } from '../../../environments/environment.development';
import { JK } from '../models/jk';

@Injectable({ providedIn: 'root' })
export class JkService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  currentJk = signal<JK | null>(null);

  getJKById(id: string) {
    return this.http.get<JK>(`${this.baseUrl}/jks/${id}`).pipe(
      tap((jk) => this.currentJk.set(jk)),
      catchError(() => {
        this.currentJk.set(null);
        return of(null);
      }),
    );
  }
}
