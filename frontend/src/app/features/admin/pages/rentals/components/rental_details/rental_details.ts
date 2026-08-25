import { Component, input, output } from '@angular/core';
import { DatePipe } from '@angular/common';
import { Rental } from '../../../../../../core/models/rental';

@Component({
  selector: 'app-admin-rental-details',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './rental_details.html',
})
export class AdminRentalDetails {
  rental = input.required<Rental>();

  closed = output<void>();
  cancelled = output<Rental>();
  forceReleased = output<Rental>();

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
}
