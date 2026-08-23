import { Component, computed, inject, signal } from '@angular/core';

import { DashboardPeriod, RevenuePoint } from '../../../../../../core/models/dashboard';

import { DashboardService } from '../../../../../../core/services/dashboard';

@Component({
  selector: 'app-admin-revenue-chart',
  standalone: true,
  templateUrl: './revenue_chart.html',
})
export class AdminRevenueChart {
  private dashboardService = inject(DashboardService);

  period = signal<DashboardPeriod>('30d');

  points = signal<RevenuePoint[]>([]);

  loading = signal(true);

  error = signal(false);

  maxAmount = computed(() => {
    const points = this.points();

    if (!points.length) {
      return 0;
    }

    return Math.max(...points.map((point) => point.amount));
  });

  total = computed(() => {
    return this.points().reduce((sum, point) => sum + point.amount, 0);
  });

  chartPoints = computed(() => {
    const points = this.points();

    if (!points.length) {
      return '';
    }

    const width = 700;
    const height = 280;

    const paddingX = 20;
    const paddingY = 20;

    const max = this.maxAmount();

    if (max === 0) {
      return '';
    }

    return points
      .map((point, index) => {
        const x =
          points.length === 1
            ? width / 2
            : paddingX + (index / (points.length - 1)) * (width - paddingX * 2);

        const y = height - paddingY - (point.amount / max) * (height - paddingY * 2);

        return `${x},${y}`;
      })
      .join(' ');
  });

  areaPoints = computed(() => {
    const line = this.chartPoints();

    if (!line) {
      return '';
    }

    return `20,280 ${line} 680,280`;
  });

  constructor() {
    this.loadRevenue();
  }

  selectPeriod(period: DashboardPeriod): void {
    if (this.period() === period) {
      return;
    }

    this.period.set(period);

    this.loadRevenue();
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value);
  }

  formatDate(date: string): string {
    return new Intl.DateTimeFormat('ru-RU', {
      day: 'numeric',
      month: 'short',
    }).format(new Date(date));
  }

  private loadRevenue(): void {
    this.loading.set(true);
    this.error.set(false);

    this.dashboardService.getRevenue(this.period()).subscribe({
      next: (data) => {
        this.points.set(data.points);
        this.loading.set(false);
      },

      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }
}
