import { Component, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { Rental, RentalStatus } from '../../../../../core/models/rental';
import { RentalService } from '../../../../../core/services/rental';

@Component({
  selector: 'app-client-rental-details',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './rental-details.html',
})
export class ClientRentalDetails {
  private route = inject(ActivatedRoute);
  private rentalService = inject(RentalService);

  loading = signal(true);
  error = signal(false);
  rental = signal<Rental | null>(null);

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
      month: 'long',
      year: 'numeric',
    }).format(new Date(value));
  }

  canCancel(): boolean {
    const currentRental = this.rental();

    return currentRental?.status === 'active' || currentRental?.status === 'locked';
  }

  cancel(): void {
    const currentRental = this.rental();

    if (!currentRental || !this.canCancel()) {
      return;
    }

    this.rentalService.cancel(currentRental.id).subscribe({
      next: (updatedRental) => {
        this.rental.set(updatedRental);
      },
      error: () => {
        this.error.set(true);
      },
    });
  }

  private load(): void {
    const rentalId = this.route.snapshot.paramMap.get('id');

    if (!rentalId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.rentalService.getById(rentalId).subscribe({
      next: (rental) => {
        this.rental.set(rental);
        this.loading.set(false);
      },
      error: () => {
        this.rental.set(null);
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }
}
