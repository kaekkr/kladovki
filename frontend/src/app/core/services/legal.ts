import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, shareReplay } from 'rxjs';

import { environment } from '../../../environments/environment';
import { Agreement, AgreementDoc, AgreementLang, AgreementStatus } from '../models/legal';

/** The user agreements (resident / ОСИ) and consent records. Texts are cached per doc+lang. */
@Injectable({ providedIn: 'root' })
export class LegalService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;
  private cache = new Map<string, Observable<Agreement>>();

  get(doc: AgreementDoc, lang: AgreementLang = 'ru'): Observable<Agreement> {
    const key = `${doc}:${lang}`;
    if (!this.cache.has(key)) {
      this.cache.set(
        key,
        this.http.get<Agreement>(`${this.baseUrl}/legal/${doc}`, { params: { lang } }).pipe(shareReplay(1)),
      );
    }
    return this.cache.get(key)!;
  }

  /** Current version — sent with registration / payment / renewal as proof of which text was accepted. */
  version(doc: AgreementDoc): Observable<string> {
    return this.get(doc).pipe(map((a) => a.version));
  }

  status() {
    return this.http.get<AgreementStatus>(`${this.baseUrl}/legal/status`);
  }

  accept(doc: AgreementDoc, version: string) {
    return this.http.post<{ message: string }>(`${this.baseUrl}/legal/accept`, { doc, version });
  }
}
