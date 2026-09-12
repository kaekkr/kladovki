import { Component, computed, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { RouterLink } from '@angular/router';

type NotificationType = 'payment' | 'rental' | 'system';

interface ClientNotification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  created_at: string;
  read: boolean;
}

@Component({
  selector: 'app-client-notifications',
  standalone: true,
  imports: [RouterLink, DatePipe],
  templateUrl: './notifications.html',
})
export class ClientNotifications {
  notifications = signal<ClientNotification[]>([
    {
      id: '1',
      type: 'payment',
      title: 'Оплата аренды',
      message: 'Платёж за кладовку успешно обработан.',
      created_at: new Date().toISOString(),
      read: false,
    },
    {
      id: '2',
      type: 'rental',
      title: 'Аренда активирована',
      message: 'Ваша аренда кладовки успешно активирована.',
      created_at: new Date(Date.now() - 86400000).toISOString(),
      read: true,
    },
  ]);

  unreadCount = computed(() => this.notifications().filter((n) => !n.read).length);

  typeIcon(type: NotificationType): string {
    switch (type) {
      case 'payment':
        return 'fa-wallet';
      case 'rental':
        return 'fa-box';
      case 'system':
        return 'fa-bell';
    }
  }

  typeClass(type: NotificationType): string {
    switch (type) {
      case 'payment':
        return 'bg-brand-accent/10 text-brand-accent';
      case 'rental':
        return 'bg-status-rented/10 text-status-rented';
      case 'system':
        return 'bg-brand-elevated text-brand-muted';
    }
  }

  markAsRead(id: string): void {
    this.notifications.update((items) =>
      items.map((item) => (item.id === id ? { ...item, read: true } : item)),
    );
  }

  markAllAsRead(): void {
    this.notifications.update((items) => items.map((item) => ({ ...item, read: true })));
  }
}
