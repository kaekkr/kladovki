import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { Router } from '@angular/router';

import { AppNotification } from '../../../core/models/notification';
import { NotificationService } from '../../../core/services/notification';
import {
  notificationIcon,
  notificationTone,
  toneClass,
} from '../../../core/utils/notification-display';

/** Notification feed used by both the resident app and the OSI panel. */
@Component({
  selector: 'app-notification-list',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './notification-list.html',
})
export class NotificationList {
  private service = inject(NotificationService);
  private router = inject(Router);

  loading = signal(true);
  error = signal(false);
  items = signal<AppNotification[]>([]);
  unreadCount = computed(() => this.items().filter((n) => !n.read).length);

  icon = notificationIcon;
  tone = (n: AppNotification) => toneClass[notificationTone(n.kind)];

  constructor() {
    this.service.list().subscribe({
      next: (list) => {
        this.items.set(list);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }

  open(n: AppNotification): void {
    if (!n.read) {
      this.items.update((list) => list.map((x) => (x.id === n.id ? { ...x, read: true } : x)));
      this.service.markRead(n.id).subscribe();
    }
    if (n.link) {
      this.router.navigateByUrl(n.link);
    }
  }

  markAll(): void {
    this.items.update((list) => list.map((x) => ({ ...x, read: true })));
    this.service.markAllRead().subscribe();
  }
}
