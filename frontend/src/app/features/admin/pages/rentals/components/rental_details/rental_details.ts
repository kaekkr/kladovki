import { Component, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DatePipe } from '@angular/common';
import { Rental } from '../../../../../../core/models/rental';

@Component({
  selector: 'app-admin-rental-details',
  standalone: true,
  imports: [DatePipe, FormsModule],
  templateUrl: './rental_details.html',
})
export class AdminRentalDetails {
  rental = input.required<Rental>();

  closed = output<void>();
  cancelled = output<Rental>();
  forceReleased = output<Rental>();
  overdueReleased = output<{ rental: Rental; record: TerminationRecord }>();

  // Termination record form (agreement п. 5.2, ОСИ regulation п. 4.2–4.4)
  releaseFormOpen = signal(false);
  reason = signal('');
  note = signal('');
  police = signal(false);
  costs = signal(0);
  formError = signal<string | null>(null);

  close(): void {
    this.closed.emit();
  }

  statusLabel(status: Rental['status']): string {
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

  remainingDays(): number {
    const rental = this.rental();

    if (rental.status !== 'active') {
      return 0;
    }

    const end = new Date(rental.ends_at).getTime();
    const now = Date.now();

    const diff = end - now;

    if (diff <= 0) {
      return 0;
    }

    return Math.ceil(diff / (1000 * 60 * 60 * 24));
  }

  cancelRental(): void {
    this.cancelled.emit(this.rental());
  }

  forceRelease(): void {
    this.forceReleased.emit(this.rental());
  }

  overdueDays() {
    const diff = Date.now() - new Date(this.rental().ends_at).getTime();

    return diff > 0 ? Math.floor(diff / 86400000) : 0;
  }

  isOverdue() {
    return this.rental().status === 'active' && this.overdueDays() > 0;
  }

  canReleaseOverdue() {
    return this.rental().status === 'active' && this.overdueDays() > 14;
  }

  releaseOverdue() {
    if (!this.reason().trim()) {
      this.formError.set('Укажите причину — арендатор увидит её');
      return;
    }
    if (this.costs() < 0) {
      this.formError.set('Расходы не могут быть отрицательными');
      return;
    }
    this.formError.set(null);
    this.overdueReleased.emit({
      rental: this.rental(),
      record: { reason: this.reason().trim(), note: this.note().trim(), police: this.police(), costs: Math.round(this.costs() || 0) },
    });
  }
}

export interface TerminationRecord {
  reason: string;
  note: string;
  police: boolean;
  costs: number;
}
