import { Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { DatePipe } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { Rental } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';

@Component({
  selector: 'app-client-dashboard',
  standalone: true,
  imports: [RouterLink, DatePipe],
  templateUrl: './dashboard.html',
})
export class ClientDashboard {
  private auth = inject(AuthService);
  private rentalService = inject(RentalService);

  loading = signal(true);
  error = signal(false);
  rentals = signal<Rental[]>([]);

  userName = computed(() => this.auth.currentUser()?.full_name || 'Житель');

  activeRentals = computed(() => this.rentals().filter((r) => r.status === 'active'));

  lockedRentals = computed(() => this.rentals().filter((r) => r.status === 'locked'));

  hasDebt = computed(() =>
    this.rentals().some(
      (r) =>
        (r.status === 'active' || r.status === 'locked') &&
        r.total_paid < r.price_per_month * r.months,
    ),
  );

  constructor() {
    this.load();
  }

  private load(): void {
    const user = this.auth.currentUser();
    const jkId = user?.jk_id;
    const userId = user?.id;

    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    // Temporary: filter admin list by user.
    // Replace with GET /rentals/me when backend is ready.
    this.rentalService.listByJK(jkId).subscribe({
      next: (all) => {
        const mine = userId ? all.filter((r) => r.user_id === userId) : [];
        this.rentals.set(mine);
        this.loading.set(false);
      },
      error: () => {
        this.rentals.set([]);
        this.loading.set(false);
        // soft-fail: empty dashboard is fine without client API yet
      },
    });
  }

  statusLabel(status: string): string {
    switch (status) {
      case 'active':
        return 'Активна';
      case 'locked':
        return 'Ожидает оплаты';
      case 'expired':
        return 'Истекла';
      case 'cancelled':
        return 'Отменена';
      default:
        return status;
    }
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }
}
