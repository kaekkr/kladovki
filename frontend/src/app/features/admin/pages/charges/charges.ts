import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { Charge, ChargeStatus } from '../../../../core/models/charge';
import { ChargeService } from '../../../../core/services/charge';

@Component({
  selector: 'app-admin-charges',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './charges.html',
})
export class AdminCharges {
  private authService = inject(AuthService);
  private chargeService = inject(ChargeService);

  charges = signal<Charge[]>([]);
  loading = signal(true);
  error = signal(false);
  search = signal('');
  statusFilter = signal<ChargeStatus | 'all'>('all');

  filteredCharges = computed(() => {
    const q = this.search().trim().toLowerCase();
    const status = this.statusFilter();
    return this.charges().filter((c) => {
      const matchesSearch =
        !q ||
        c.storage_number.toLowerCase().includes(q) ||
        c.user_name.toLowerCase().includes(q) ||
        c.user_phone.toLowerCase().includes(q);
      const matchesStatus = status === 'all' || c.status === status;
      return matchesSearch && matchesStatus;
    });
  });

  totals = computed(() => {
    const list = this.filteredCharges();
    return {
      accrued: list.reduce((s, c) => s + c.accrued, 0),
      paid: list.reduce((s, c) => s + c.paid, 0),
      balance: list.reduce((s, c) => s + c.balance, 0),
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
    this.chargeService.listByJK(jkId).subscribe({
      next: (items) => {
        this.charges.set(items);
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
      case 'paid':
        return 'Оплачено';
      case 'partial':
        return 'Частично';
      case 'pending':
        return 'Ожидает';
      case 'overdue':
        return 'Просрочено';
      default:
        return status;
    }
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }
}
