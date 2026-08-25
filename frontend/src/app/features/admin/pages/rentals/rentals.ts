import { Component, computed, inject, signal } from '@angular/core';

import { AuthService } from '../../../../core/auth/auth.service';
import { Rental, RentalStatus } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';
import { DatePipe } from '@angular/common';
import { AdminRentalDetails } from './components/rental_details/rental_details';

@Component({
  selector: 'app-admin-rentals',
  standalone: true,
  imports: [DatePipe, AdminRentalDetails],
  templateUrl: './rentals.html',
})
export class AdminRentals {
  private authService = inject(AuthService);
  private rentalService = inject(RentalService);

  rentals = signal<Rental[]>([]);

  loading = signal(true);
  error = signal(false);

  search = signal('');
  statusFilter = signal<RentalStatus | 'all'>('all');

  selectedRental = signal<Rental | null>(null);

  actionError = signal('');

  filteredRentals = computed(() => {
    const search = this.search().trim().toLowerCase();
    const status = this.statusFilter();

    return this.rentals().filter((rental) => {
      const matchesSearch =
        !search ||
        rental.storage_number.toLowerCase().includes(search) ||
        rental.user_name.toLowerCase().includes(search) ||
        rental.user_phone.toLowerCase().includes(search);

      const matchesStatus = status === 'all' || rental.status === status;

      return matchesSearch && matchesStatus;
    });
  });

  constructor() {
    this.loadRentals();
  }

  private loadRentals(): void {
    const jkId = this.authService.currentUser()?.jk_id;

    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.rentalService.listByJK(jkId).subscribe({
      next: (rentals) => {
        this.rentals.set(rentals);
        this.loading.set(false);
      },

      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  setSearch(value: string): void {
    this.search.set(value);
  }

  setStatusFilter(status: RentalStatus | 'all'): void {
    this.statusFilter.set(status);
  }

  openRental(rental: Rental): void {
    this.selectedRental.set(rental);
  }

  closeRental(): void {
    this.selectedRental.set(null);
  }

  cancelRental(rental: Rental): void {
    if (!confirm(`Досрочно завершить аренду кладовки №${rental.storage_number}?`)) {
      return;
    }

    this.actionError.set('');

    this.rentalService.cancel(rental.id).subscribe({
      next: (updatedRental) => {
        this.rentals.update((rentals) =>
          rentals.map((item) => (item.id === updatedRental.id ? updatedRental : item)),
        );

        this.selectedRental.set(null);
      },

      error: (error) => {
        this.actionError.set(error?.error?.error || 'Не удалось завершить аренду.');
      },
    });
  }

  forceReleaseRental(rental: Rental): void {
    if (!confirm(`Снять блокировку с кладовки №${rental.storage_number}?`)) {
      return;
    }
    this.actionError.set('');
    this.rentalService.forceRelease(rental.id).subscribe({
      next: (updated) => {
        this.rentals.update((list) =>
          list.map((item) => (item.id === updated.id ? updated : item)),
        );
        this.selectedRental.set(null);
      },
      error: (err) => {
        this.actionError.set(err?.error?.error || 'Не удалось снять блокировку.');
      },
    });
  }

  statusLabel(status: RentalStatus): string {
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
