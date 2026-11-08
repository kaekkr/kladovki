import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';

import { environment } from '../../../environments/environment';
import { downloadFile } from '../utils/csv';

export type ReportKind = 'payments' | 'storages' | 'charges';

/** ОСИ reports (regulation п. 5.1) as Excel-friendly CSV files. */
@Injectable({ providedIn: 'root' })
export class ReportService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  download(jkId: string, kind: ReportKind, period?: { from: string; to: string }): Observable<void> {
    const params: Record<string, string> = {};
    if (period?.from) params['from'] = period.from;
    if (period?.to) params['to'] = period.to;
    const suffix = period ? `_${period.from || 'начало'}_${period.to || 'сегодня'}` : '';
    const names: Record<ReportKind, string> = {
      payments: `Поступления${suffix}.csv`,
      storages: 'Кладовки.csv',
      charges: 'Начисления.csv',
    };
    return this.http
      .get(`${this.baseUrl}/reports/jk/${jkId}/${kind}.csv`, { params, responseType: 'blob' })
      .pipe(map((blob) => downloadFile(blob, names[kind])));
  }
}
