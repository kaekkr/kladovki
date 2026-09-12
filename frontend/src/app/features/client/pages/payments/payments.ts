import { Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { Payment, PaymentStatus } from '../../../../core/models/payment';
import { PaymentService } from '../../../../core/services/payment';

@Component({
  selector: 'app-client-payments',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './payments.html',
})
export class ClientPayments {
  private auth = inject(AuthService);
  private paymentService = inject(PaymentService);

  loading = signal(true);
  error = signal(false);

  payments = signal<Payment[]>([]);

  totalPaid = computed(() =>
    this.payments()
      .filter((payment) => payment.status === 'success')
      .reduce((sum, payment) => sum + payment.amount, 0),
  );

  successfulCount = computed(
    () => this.payments().filter((payment) => payment.status === 'success').length,
  );

  pendingCount = computed(
    () => this.payments().filter((payment) => payment.status === 'pending').length,
  );

  constructor() {
    this.load();
  }

  statusLabel(status: PaymentStatus): string {
    switch (status) {
      case 'success':
        return 'Оплачено';

      case 'pending':
        return 'В обработке';

      case 'failed':
        return 'Не оплачено';

      default:
        return status;
    }
  }

  statusClass(status: PaymentStatus): string {
    switch (status) {
      case 'success':
        return 'bg-brand-accent/10 text-brand-accent';

      case 'pending':
        return 'bg-status-locked/10 text-status-locked';

      case 'failed':
        return 'bg-status-overdue/10 text-status-overdue';

      default:
        return 'bg-brand-elevated text-brand-muted';
    }
  }

  statusIcon(status: PaymentStatus): string {
    switch (status) {
      case 'success':
        return 'fa-check';

      case 'pending':
        return 'fa-clock';

      case 'failed':
        return 'fa-xmark';

      default:
        return 'fa-credit-card';
    }
  }

  providerLabel(provider: string): string {
    switch (provider.toLowerCase()) {
      case 'kaspi':
        return 'Kaspi';

      case 'card':
        return 'Банковская карта';

      default:
        return provider;
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

  formatTime(value: string): string {
    return new Intl.DateTimeFormat('ru-RU', {
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(value));
  }

  trackByPayment(_: number, payment: Payment): string {
    return payment.id;
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

    this.paymentService.listByJK(jkId).subscribe({
      next: (allPayments) => {
        const myPayments = allPayments
          .filter((payment) => payment.user_id === userId)
          .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

        this.payments.set(myPayments);
        this.loading.set(false);
      },

      error: () => {
        this.payments.set([]);
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }
}
