import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { tap } from 'rxjs';

import { AuthService } from '../../../../core/auth/auth.service';
import { environment } from '../../../../../environments/environment';

/**
 * The resident's own phone (also the default Kaspi number for invoices).
 * `PUT /me/phone {phone}` with the canonical `+77XXXXXXXXX`; the server
 * answers with the updated user or `{message}`.
 */
@Injectable({ providedIn: 'root' })
export class ProfilePhoneService {
  private http = inject(HttpClient);
  private auth = inject(AuthService);

  updatePhone(phone: string) {
    return this.http
      .put<{ phone?: string; message?: string }>(`${environment.apiUrl}/me/phone`, { phone })
      .pipe(
        tap((res) => {
          const user = this.auth.currentUser();
          if (user) this.auth.currentUser.set({ ...user, phone: res?.phone || phone });
        }),
      );
  }
}
