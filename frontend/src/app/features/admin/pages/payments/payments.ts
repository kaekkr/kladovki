import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe, UpperCasePipe } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { Payment, PaymentStatus } from '../../../../core/models/payment';
import { PaymentService } from '../../../../core/services/payment';
import { ReportService } from '../../../../core/services/report';

@Component({
  selector: 'app-admin-payments',
  standalone: true,
  imports: [DatePipe, UpperCasePipe],
  templateUrl: './payments.html',
})
export class AdminPayments {
  private authService = inject(AuthService);
  private paymentService = inject(PaymentService);
  private reports = inject(ReportService);

  // Period (regulation п. 5.1: reports for any period). Default: this month.
  from = signal(monthStart());
  to = signal(today());
  downloading = signal(false);

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

  /** Income report (payments + refunds paid out) for the selected period, as an Excel file. */
  downloadReport(): void {
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) return;
    this.downloading.set(true);
    this.reports.download(jkId, 'payments', { from: this.from(), to: this.to() }).subscribe({
      next: () => this.downloading.set(false),
      error: () => this.downloading.set(false),
    });
  }

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
    this.paymentService.listByJK(jkId, { from: this.from(), to: this.to() }).subscribe({
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

const KZ_OFFSET_MS = 5 * 3600 * 1000;
function today(): string {
  return new Date(Date.now() + KZ_OFFSET_MS).toISOString().slice(0, 10);
}
function monthStart(): string {
  return today().slice(0, 8) + '01';
}
