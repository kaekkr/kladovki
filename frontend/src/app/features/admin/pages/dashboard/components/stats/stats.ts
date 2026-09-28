import { Component, computed, inject, signal } from '@angular/core';
import { DashboardService } from '../../../../../../core/services/dashboard';
import { DashboardPeriod, DashboardStats } from '../../../../../../core/models/dashboard';

@Component({
  selector: 'app-admin-stats',
  standalone: true,
  templateUrl: './stats.html',
})
export class AdminStats {
  Math = Math;

  private dashboardService = inject(DashboardService);

  selectedPeriod = signal<DashboardPeriod>('30d');

  stats = signal<DashboardStats | null>(null);

  loading = signal(true);

  error = signal(false);

  occupancyPercent = computed(() => {
    const stats = this.stats();

    if (!stats || !stats.total_storages || stats.total_storages === 0) {
      return 0;
    }

    return Math.round((stats.occupied_storages / stats.total_storages) * 100);
  });

  constructor() {
    this.loadStats();
  }

  selectPeriod(period: DashboardPeriod): void {
    if (this.selectedPeriod() === period) {
      return;
    }

    this.selectedPeriod.set(period);
    this.loadStats();
  }

  private loadStats(): void {
    this.loading.set(true);
    this.error.set(false);

    this.dashboardService.getStats(this.selectedPeriod()).subscribe({
      next: (stats) => {
        console.log('API Stats response:', stats); // <-- Добавьте это
        this.stats.set(stats);
        this.loading.set(false);
      },

      error: (err) => {
        console.error('API Stats error:', err);
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value);
  }
}
