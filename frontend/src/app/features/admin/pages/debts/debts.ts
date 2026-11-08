import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { ChargeService } from '../../../../core/services/charge';
import { chargeStatusLabel } from '../../../../core/utils/charge-display';
import { AuthService } from '../../../../core/auth/auth.service';
import { ReportService } from '../../../../core/services/report';
import { Charge, ChargeStatus } from '../../../../core/models/charge';

@Component({
  selector: 'app-admin-debts',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './debts.html',
})
export class AdminDebts {
  private authService = inject(AuthService);
  private reports = inject(ReportService);
  downloading = signal(false);

  /** Excel-friendly report (ОСИ regulation п. 5.1). */
  downloadReport(): void {
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) return;
    this.downloading.set(true);
    this.reports.download(jkId, 'charges').subscribe({
      next: () => this.downloading.set(false),
      error: () => this.downloading.set(false),
    });
  }
  private chargeService = inject(ChargeService);

  debts = signal<Charge[]>([]);
  loading = signal(true);
  error = signal(false);
  search = signal('');
  statusFilter = signal<ChargeStatus | 'all'>('all');

  filteredDebts = computed(() => {
    const q = this.search().trim().toLowerCase();
    const status = this.statusFilter();
    return this.debts().filter((d) => {
      const matchesSearch =
        !q ||
        d.storage_number.toLowerCase().includes(q) ||
        d.user_name.toLowerCase().includes(q) ||
        d.user_phone.toLowerCase().includes(q);
      const matchesStatus = status === 'all' || d.status === status;
      return matchesSearch && matchesStatus;
    });
  });

  totals = computed(() => {
    const list = this.filteredDebts();
    return {
      count: list.length,
      balance: list.reduce((s, d) => s + d.balance, 0),
      accrued: list.reduce((s, d) => s + d.accrued, 0),
      paid: list.reduce((s, d) => s + d.paid, 0),
    };
  });

  constructor() {
    this.load();
  }

  private load(): void {
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }
    this.loading.set(true);
    this.error.set(false);
    this.chargeService.listDebtsByJK(jkId).subscribe({
      next: (items) => {
        this.debts.set(items);
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

  setStatusFilter(status: ChargeStatus | 'all'): void {
    this.statusFilter.set(status);
  }

  statusLabel = chargeStatusLabel;

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }
}
