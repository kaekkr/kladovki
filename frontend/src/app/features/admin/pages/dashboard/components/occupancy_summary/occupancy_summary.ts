import { Component, computed, inject, signal } from '@angular/core';
import { DashboardOccupancy } from '../../../../../../core/models/dashboard';
import { DashboardService } from '../../../../../../core/services/dashboard';

@Component({
  selector: 'app-admin-occupancy-summary',
  standalone: true,
  templateUrl: './occupancy_summary.html',
})
export class AdminOccupancySummary {
  private dashboardService = inject(DashboardService);

  occupancy = signal<DashboardOccupancy | null>(null);

  loading = signal(true);
  error = signal(false);

  occupiedPercent = computed(() => {
    const data = this.occupancy();

    if (!data || data.total === 0) {
      return 0;
    }

    return Math.round((data.occupied / data.total) * 100);
  });

  freePercent = computed(() => {
    const data = this.occupancy();

    if (!data || data.total === 0) {
      return 0;
    }

    return Math.round((data.free / data.total) * 100);
  });

  lockedPercent = computed(() => {
    const data = this.occupancy();

    if (!data || data.total === 0) {
      return 0;
    }

    return Math.round((data.locked / data.total) * 100);
  });

  constructor() {
    this.loadOccupancy();
  }

  private loadOccupancy(): void {
    this.loading.set(true);
    this.error.set(false);

    this.dashboardService.getOccupancy().subscribe({
      next: (occupancy) => {
        this.occupancy.set(occupancy);
        this.loading.set(false);
      },

      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }
}
