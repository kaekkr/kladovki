import { Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { NotificationService } from '../../../../core/services/notification';
import { NotificationChannels } from '../../../../core/models/notification';

@Component({
  selector: 'app-client-profile',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './profile.html',
})
export class ClientProfile {
  private auth = inject(AuthService);
  private router = inject(Router);

  private notifications = inject(NotificationService);

  user = computed(() => this.auth.currentUser());
  channels = signal<NotificationChannels | null>(null);
  linking = signal(false);
  linkHint = signal('');

  constructor() {
    this.loadChannels();
  }

  private loadChannels(): void {
    this.notifications.channels().subscribe({
      next: (c) => this.channels.set(c),
      error: () => this.channels.set(null),
    });
  }

  linkTelegram(): void {
    this.linking.set(true);
    this.notifications.telegramLink().subscribe({
      next: ({ url }) => {
        this.linking.set(false);
        window.open(url, '_blank');
        this.linkHint.set('Нажмите «Start» в Telegram, затем вернитесь сюда — статус обновится.');
        // Re-check a few times while the user confirms in Telegram.
        let tries = 0;
        const timer = setInterval(() => {
          tries++;
          this.loadChannels();
          if (this.channels()?.telegram || tries > 20) {
            clearInterval(timer);
            if (this.channels()?.telegram) this.linkHint.set('');
          }
        }, 3000);
      },
      error: (err) => {
        this.linking.set(false);
        this.linkHint.set(err?.error?.error || 'Не удалось получить ссылку');
      },
    });
  }

  unlinkTelegram(): void {
    this.notifications.telegramUnlink().subscribe(() => this.loadChannels());
  }

  initials = computed(() => {
    const name = this.user()?.full_name?.trim();

    if (!name) {
      return 'Ж';
    }

    return name
      .split(/\s+/)
      .slice(0, 2)
      .map((part) => part.charAt(0))
      .join('')
      .toUpperCase();
  });

  logout(): void {
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}
