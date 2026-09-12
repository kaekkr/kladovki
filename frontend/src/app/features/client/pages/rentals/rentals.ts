import { Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { Rental, RentalStatus } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';

@Component({
  selector: 'app-client-rentals',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './rentals.html',
})
export class ClientRentals {
  private auth = inject(AuthService);
  private rentalService = inject(RentalService);

  loading = signal(true);
  error = signal(false);
  rentals = signal<Rental[]>([]);

  activeRentals = computed(() => this.rentals().filter((rental) => rental.status === 'active'));

  historyRentals = computed(() =>
    this.rentals().filter((rental) => ['expired', 'cancelled'].includes(rental.status)),
  );

  constructor() {
    this.load();
  }

  statusLabel(status: RentalStatus): string {
    switch (status) {
      case 'locked':
        return 'Оформляется';
      case 'active':
        return 'Активна';
      case 'expired':
        return 'Завершена';
      case 'cancelled':
        return 'Отменена';
      default:
        return status;
    }
  }

  statusClass(status: RentalStatus): string {
    switch (status) {
      case 'active':
        return 'bg-brand-accent/10 text-brand-accent';

      case 'locked':
        return 'bg-status-locked/10 text-status-locked';

      case 'expired':
      case 'cancelled':
        return 'bg-brand-elevated text-brand-muted';

      default:
        return 'bg-brand-elevated text-brand-muted';
    }
  }

  formatMoney(value: number): string {
    return (
      new Intl.NumberFormat('ru-RU', {
        maximumFractionDigits: 0,
      }).format(value) + ' ₸'
    );
  }

  formatDate(value: string): string {
    return new Intl.DateTimeFormat('ru-RU', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    }).format(new Date(value));
  }

  private load(): void {
    const user = this.auth.currentUser();
    const jkId = user?.jk_id;
    const userId = user?.id;

    if (!jkId || !userId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.rentalService.listByJK(jkId).subscribe({
      next: (allRentals) => {
        const myRentals = allRentals.filter((rental) => rental.user_id === userId);

        this.rentals.set(myRentals);
        this.loading.set(false);
      },

      error: () => {
        this.rentals.set([]);
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }
}
