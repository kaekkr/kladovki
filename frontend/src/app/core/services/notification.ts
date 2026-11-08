import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { tap } from 'rxjs';

import { environment } from '../../../environments/environment';
import { AppNotification, NotificationChannels } from '../models/notification';

@Injectable({ providedIn: 'root' })
export class NotificationService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  /** Shared unread counter for header bells. */
  unread = signal(0);

  list() {
    return this.http.get<AppNotification[]>(`${this.baseUrl}/notifications`);
  }

  refreshUnread(): void {
    this.http.get<{ unread: number }>(`${this.baseUrl}/notifications/unread-count`).subscribe({
      next: (r) => this.unread.set(r.unread),
      error: () => {},
    });
  }

  markRead(id: string) {
    return this.http
      .post(`${this.baseUrl}/notifications/${id}/read`, {})
      .pipe(tap(() => this.refreshUnread()));
  }

  markAllRead() {
    return this.http
      .post(`${this.baseUrl}/notifications/read-all`, {})
      .pipe(tap(() => this.unread.set(0)));
  }

  channels() {
    return this.http.get<NotificationChannels>(`${this.baseUrl}/me/channels`);
  }

  telegramLink() {
    return this.http.post<{ url: string }>(`${this.baseUrl}/me/telegram/link`, {});
  }

  telegramUnlink() {
    return this.http.delete(`${this.baseUrl}/me/telegram`);
  }
}
