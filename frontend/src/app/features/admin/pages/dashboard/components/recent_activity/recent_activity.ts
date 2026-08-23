import { Component, inject, signal } from '@angular/core';
import { DashboardActivity } from '../../../../../../core/models/dashboard';
import { DashboardService } from '../../../../../../core/services/dashboard';

@Component({
  selector: 'app-admin-recent-activity',
  standalone: true,
  templateUrl: './recent_activity.html',
})
export class AdminRecentActivity {
  private dashboardService = inject(DashboardService);

  activities = signal<DashboardActivity[]>([]);
  loading = signal(true);
  error = signal(false);

  constructor() {
    this.loadActivity();
  }

  private loadActivity(): void {
    this.loading.set(true);
    this.error.set(false);

    this.dashboardService.getActivity().subscribe({
      next: (activities) => {
        this.activities.set(activities ?? []);
        this.loading.set(false);
      },

      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value);
  }

  formatTime(value: string): string {
    const date = new Date(value);

    return new Intl.DateTimeFormat('ru-RU', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    }).format(date);
  }
}
