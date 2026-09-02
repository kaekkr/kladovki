import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, forkJoin, map, catchError } from 'rxjs';
import { environment } from '../../../environments/environment.development';
import { AdminSettings, JKSettings, TariffSettings, RentalSettings } from '../models/settings';

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  /** Load everything for the current JK */
  load(jkId: string): Observable<AdminSettings> {
    return forkJoin({
      jk: this.getJK(jkId),
      tariff: this.getTariff(jkId),
      rental: this.getRentalSettings(jkId),
    });
  }

  getJK(jkId: string): Observable<JKSettings> {
    return this.http
      .get<JKSettings>(`${this.baseUrl}/jks/${jkId}`)
      .pipe(catchError(() => of({ id: jkId, name: 'Мой ЖК', address: '' })));
  }

  getTariff(jkId: string): Observable<TariffSettings> {
    // Adjust path if your API differs
    return this.http.get<{ amount: number }>(`${this.baseUrl}/jks/${jkId}/tariff`).pipe(
      map((t) => ({ jk_id: jkId, amount: t.amount ?? 0 })),
      catchError(() => of({ jk_id: jkId, amount: 0 })),
    );
  }

  getRentalSettings(jkId: string): Observable<RentalSettings> {
    // No backend yet → defaults
    return of({ lock_duration_minutes: 2 });
  }

  saveTariff(jkId: string, amount: number): Observable<void> {
    return this.http
      .put<void>(`${this.baseUrl}/jks/${jkId}/tariff`, { amount })
      .pipe(catchError(() => of(void 0)));
  }

  saveRentalSettings(jkId: string, settings: RentalSettings): Observable<void> {
    // Wire when backend exists
    return of(void 0);
  }
}
