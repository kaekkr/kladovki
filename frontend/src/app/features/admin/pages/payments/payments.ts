import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe, UpperCasePipe } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { Payment, PaymentStatus } from '../../../../core/models/payment';
import { PaymentService } from '../../../../core/services/payment';

@Component({
  selector: 'app-admin-payments',
  standalone: true,
  imports: [DatePipe, UpperCasePipe],
  templateUrl: './payments.html',
})
export class AdminPayments {
  private authService = inject(AuthService);
  private paymentService = inject(PaymentService);

  payments = signal<Payment[]>([]);
  loading = signal(true);
  error = signal(false);

  search = signal('');
  statusFilter = signal<PaymentStatus | 'all'>('all');

  // Filtered list based on search and status
  filteredPayments = computed(() => {
    const search = this.search().trim().toLowerCase();
    const status = this.statusFilter();

    return this.payments().filter((p) => {
      const matchesSearch =
        !search ||
        p.storage_number.toLowerCase().includes(search) ||
        p.user_name.toLowerCase().includes(search);
      const matchesStatus = status === 'all' || p.status === status;
      return matchesSearch && matchesStatus;
    });
  });

  // Calculate total successful revenue
  totalRevenue = computed(() => {
    return this.payments()
      .filter((p) => p.status === 'success')
      .reduce((sum, p) => sum + p.amount, 0);
  });

  constructor() {
    this.loadPayments();
  }

  loadPayments(): void {
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) {
      this.loading.set(false);
      return;
    }

    this.loading.set(true);
    this.paymentService.listByJK(jkId).subscribe({
      next: (data) => {
        this.payments.set(data);
        this.loading.set(false);
      },
      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  formatMoney(value: number): string {
    // If your backend stores in Tiyn (cents), use value / 100
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }

  statusLabel(status: PaymentStatus): string {
    switch (status) {
      case 'success':
        return 'Оплачено';
      case 'pending':
        return 'Ожидание';
      case 'failed':
        return 'Ошибка';
      default:
        return status;
    }
  }
}
