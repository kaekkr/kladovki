import { Component, inject, OnDestroy, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { Rental, RentalStatus } from '../../../../../core/models/rental';
import { RentalService } from '../../../../../core/services/rental';

@Component({
  selector: 'app-client-rental-details',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './rental-details.html',
})
export class ClientRentalDetails implements OnDestroy {
  private route = inject(ActivatedRoute);
  private rentalService = inject(RentalService);

  loading = signal(true);
  error = signal(false);
  rental = signal<Rental | null>(null);

  paying = signal(false);
  paymentError = signal<string | null>(null);

  remainingSeconds = signal(0);

  private timer: ReturnType<typeof setInterval> | null = null;

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

  formatTime(seconds: number): string {
    const minutes = Math.floor(seconds / 60);
    const remaining = seconds % 60;

    return `${String(minutes).padStart(2, '0')}:${String(remaining).padStart(2, '0')}`;
  }

  canPay(): boolean {
    return this.rental()?.status === 'locked' && this.remainingSeconds() > 0 && !this.paying();
  }

  pay(): void {
    const currentRental = this.rental();

    if (!currentRental || !this.canPay()) {
      return;
    }

    this.paying.set(true);
    this.paymentError.set(null);

    this.rentalService.confirmPayment(currentRental.id).subscribe({
      next: (updatedRental) => {
        this.rental.set(updatedRental);
        this.paying.set(false);
        this.stopTimer();
      },
      error: (error) => {
        this.paying.set(false);

        if (error.status === 409) {
          this.paymentError.set(
            'Срок бронирования истёк. Кладовка больше не удерживается за вами.',
          );

          this.load();
          return;
        }

        this.paymentError.set('Не удалось выполнить оплату. Попробуйте ещё раз.');
      },
    });
  }

  private startTimer(endsAt: string): void {
    this.stopTimer();

    const update = () => {
      const end = new Date(endsAt).getTime();
      const now = Date.now();

      const seconds = Math.max(0, Math.ceil((end - now) / 1000));

      this.remainingSeconds.set(seconds);

      if (seconds <= 0) {
        this.stopTimer();

        if (this.rental()?.status === 'locked') {
          this.load();
        }
      }
    };

    update();

    this.timer = setInterval(update, 1000);
  }

  private stopTimer(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
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
    this.paymentError.set(null);

    this.rentalService.getById(rentalId).subscribe({
      next: (rental) => {
        this.rental.set(rental);
        this.loading.set(false);

        if (rental.status === 'locked') {
          this.startTimer(rental.ends_at);
        } else {
          this.stopTimer();
          this.remainingSeconds.set(0);
        }
      },
      error: () => {
        this.rental.set(null);
        this.loading.set(false);
        this.error.set(true);
        this.stopTimer();
      },
    });
  }

  ngOnDestroy(): void {
    this.stopTimer();
  }
}
