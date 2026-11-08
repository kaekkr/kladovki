import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, forkJoin, map, catchError } from 'rxjs';
// import { environment } from '../../../environments/environment';
import { environment } from '../../../environments/environment';
import { AdminSettings, JKSettings, TariffSettings, RentalSettings } from '../models/settings';

// Интерфейс для ответа предварительного расчёта перерасчета
export interface TariffPreviewResponse {
  affected_count: number;
  total_difference: number;
}

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
    return this.http
      .get<RentalSettings>(`${this.baseUrl}/jks/${jkId}/rental-settings`)
      .pipe(catchError(() => of({ lock_duration_minutes: 2 })));
  }

  /** Предварительный расчет при уменьшении тарифа (Preview) */
  /** Предварительный расчет при уменьшении тарифа (Preview через GET) */
  previewTariffReduction(jkId: string, newAmount: number): Observable<TariffPreviewResponse> {
    return this.http.get<TariffPreviewResponse>(`${this.baseUrl}/jks/${jkId}/tariff/preview`, {
      params: { amount: newAmount.toString() },
    });
  }

  /** Returns how many tenants were asked to choose extend/refund (tariff decrease). */
  saveTariff(jkId: string, amount: number): Observable<{ recalc_requests: number }> {
    return this.http.put<{ recalc_requests: number }>(`${this.baseUrl}/jks/${jkId}/tariff`, {
      amount,
    });
  }

  /** Lock duration: 1–60 minutes (validated on the server). */
  saveRentalSettings(jkId: string, settings: RentalSettings): Observable<RentalSettings> {
    return this.http.put<RentalSettings>(`${this.baseUrl}/jks/${jkId}/rental-settings`, settings);
  }
}
