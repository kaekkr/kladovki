import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { ChargeService } from '../../../../core/services/charge';
import { AuthService } from '../../../../core/auth/auth.service';
import { Charge, ChargeStatus } from '../../../../core/models/charge';

@Component({
  selector: 'app-admin-debts',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './debts.html',
})
export class AdminDebts {
  private authService = inject(AuthService);
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

  statusLabel(status: ChargeStatus): string {
    switch (status) {
      case 'partial':
        return 'Частично';
      case 'pending':
        return 'Ожидает';
      case 'overdue':
        return 'Просрочено';
      case 'paid':
        return 'Оплачено';
      default:
        return status;
    }
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }
}
