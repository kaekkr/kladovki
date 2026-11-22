import { Component, computed, inject, signal } from '@angular/core';

import { AuthService } from '../../../../core/auth/auth.service';
import { Rental, RentalStatus } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';
import { DatePipe } from '@angular/common';
import { AdminConfirm } from '../../components/confirm-dialog/confirm';
import { LoadError } from '../../../../shared/components/load-error/load-error';
import { matchesSearch } from '../../../../core/utils/admin-search';
import { TerminationRecord, AdminRentalDetails } from './components/rental_details/rental_details';

@Component({
  selector: 'app-admin-rentals',
  standalone: true,
  imports: [DatePipe, AdminRentalDetails, LoadError],
  templateUrl: './rentals.html',
})
export class AdminRentals {
  private authService = inject(AuthService);
  private rentalService = inject(RentalService);
  private confirm = inject(AdminConfirm);

  rentals = signal<Rental[]>([]);

  loading = signal(true);
  error = signal(false);

  search = signal('');
  statusFilter = signal<RentalStatus | 'all' | 'overdue'>('all');

  selectedRental = signal<Rental | null>(null);

  actionError = signal('');

  filteredRentals = computed(() => {
    const search = this.search().trim().toLowerCase();
    const status = this.statusFilter();

    return this.rentals().filter((rental) => {
      const found = matchesSearch([rental.storage_number, rental.user_name, rental.user_phone], search);
      const matchesStatus =
        status === 'all' ||
        (status === 'overdue' ? this.overdueDays(rental) > 0 : rental.status === status);
      return found && matchesStatus;
    });
  });

  constructor() {
    this.loadRentals();
  }

  loadRentals(): void {
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

  setStatusFilter(status: RentalStatus | 'all' | 'overdue'): void {
    this.statusFilter.set(status);
  }

  openRental(rental: Rental): void {
    this.selectedRental.set(rental);
  }

  closeRental(): void {
    this.selectedRental.set(null);
  }

  async cancelRental(rental: Rental): Promise<void> {
    const ok = await this.confirm.ask({
      title: 'Завершить аренду досрочно',
      message: `Аренда кладовки №${rental.storage_number} будет отменена, кладовка станет свободной. Деньги арендатору приложение не возвращает.`,
      confirmLabel: 'Завершить аренду',
      danger: true,
    });
    if (!ok) return;

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

  async forceReleaseRental(rental: Rental): Promise<void> {
    const ok = await this.confirm.ask({
      title: 'Снять бронь',
      message: `Неоплаченная бронь кладовки №${rental.storage_number} будет снята, кладовка станет свободной.`,
      confirmLabel: 'Снять бронь',
    });
    if (!ok) return;
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

  async releaseOverdueRental({ rental, record }: { rental: Rental; record: TerminationRecord }) {
    const ok = await this.confirm.ask({
      title: 'Расторгнуть договор',
      message: `Договор по кладовке №${rental.storage_number} будет расторгнут, кладовка станет свободной. Арендатор получит уведомление.`,
      confirmLabel: 'Расторгнуть',
      danger: true,
    });
    if (!ok) return;

    this.rentalService.releaseOverdue(rental.id, record).subscribe({
      next: (updated) => {
        this.rentals.update((items) =>
          items.map((item) => (item.id === updated.id ? updated : item)),
        );

        if (this.selectedRental()?.id === updated.id) {
          this.selectedRental.set(updated);
        }
      },
      error: (err) => {
        this.actionError.set(err.error?.error ?? 'Не удалось освободить кладовку');
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

  overdueDays(rental: Rental): number {
    if (rental.status !== 'active') return 0;
    const diff = Date.now() - new Date(rental.ends_at).getTime();
    return diff > 0 ? Math.floor(diff / 86400000) : 0;
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }
}
