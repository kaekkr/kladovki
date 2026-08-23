import { Component, inject, signal } from '@angular/core';
import { DashboardService } from '../../../../../../core/services/dashboard';
import { DashboardAttention } from '../../../../../../core/models/dashboard';

@Component({
  selector: 'app-admin-attention',
  standalone: true,
  templateUrl: './attention.html',
})
export class AdminAttention {
  private dashboardService = inject(DashboardService);

  items = signal<DashboardAttention[]>([]);

  loading = signal(true);

  error = signal(false);

  constructor() {
    this.loadAttention();
  }

  private loadAttention(): void {
    this.loading.set(true);
    this.error.set(false);

    this.dashboardService.getAttention().subscribe({
      next: (items) => {
        this.items.set(items);
        this.loading.set(false);
      },

      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  getTypeLabel(item: DashboardAttention): string {
    switch (item.type) {
      case 'rental_expiring':
        return 'Аренда заканчивается';

      case 'storage_locked':
        return 'Ожидает оплаты';

      default:
        return 'Требует внимания';
    }
  }

  getDaysLabel(days: number): string {
    if (days === 1) {
      return '1 день';
    }

    if (days >= 2 && days <= 4) {
      return `${days} дня`;
    }

    return `${days} дней`;
  }

  formatDate(value: string): string {
    return new Intl.DateTimeFormat('ru-RU', {
      day: 'numeric',
      month: 'short',
    }).format(new Date(value));
  }
}
