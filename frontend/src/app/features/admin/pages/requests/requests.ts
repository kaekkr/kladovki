import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';

import { AuthService } from '../../../../core/auth/auth.service';
import { TenantRequest } from '../../../../core/models/request';
import { RequestService } from '../../../../core/services/request';
import {
  needsOsiAction,
  requestStatusLabel,
  requestTitle,
} from '../../../../core/utils/request-display';

type Filter = 'open' | 'awaiting_choice' | 'closed' | 'all';

/** OSI request monitor: early terminations, refunds after tariff decrease, late payments. */
@Component({
  selector: 'app-admin-requests',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './requests.html',
})
export class AdminRequests {
  private auth = inject(AuthService);
  private requestService = inject(RequestService);

  requests = signal<TenantRequest[]>([]);
  loading = signal(true);
  error = signal(false);
  filter = signal<Filter>('open');

  // Resolve dialog
  active = signal<TenantRequest | null>(null);
  action = signal<'approve' | 'reject'>('approve');
  comment = signal('');
  saving = signal(false);
  actionError = signal<string | null>(null);

  title = requestTitle;
  statusLabel = requestStatusLabel;
  needsAction = needsOsiAction;

  counts = computed(() => {
    const list = this.requests();
    return {
      open: list.filter((r) => r.status === 'open').length,
      awaiting: list.filter((r) => r.status === 'awaiting_choice').length,
    };
  });

  visible = computed(() => {
    const f = this.filter();
    return this.requests().filter((r) => {
      if (f === 'all') return true;
      if (f === 'closed') return r.status === 'done' || r.status === 'rejected';
      return r.status === f;
    });
  });

  constructor() {
    this.load();
  }

  load(): void {
    const jkId = this.auth.currentUser()?.jk_id;
    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }
    this.loading.set(true);
    this.requestService.listByJK(jkId).subscribe({
      next: (list) => {
        this.requests.set(list);
        this.loading.set(false);
        this.error.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }

  open(r: TenantRequest, action: 'approve' | 'reject'): void {
    this.active.set(r);
    this.action.set(action);
    this.comment.set('');
    this.actionError.set(null);
  }

  approveLabel(r: TenantRequest): string {
    return r.type === 'early_termination' ? 'Одобрить и освободить кладовку' : 'Деньги возвращены';
  }

  submit(): void {
    const r = this.active();
    if (!r) return;
    this.saving.set(true);
    this.requestService.resolve(r.id, this.action(), this.comment().trim()).subscribe({
      next: () => {
        this.saving.set(false);
        this.active.set(null);
        this.load();
      },
      error: (err) => {
        this.saving.set(false);
        this.actionError.set(err?.error?.error || 'Не удалось обработать заявку');
      },
    });
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 0 }).format(value) + ' ₸';
  }

  statusClass(r: TenantRequest): string {
    switch (r.status) {
      case 'open':
        return 'bg-orange-500/10 text-orange-500';
      case 'awaiting_choice':
        return 'bg-status-locked/10 text-status-locked';
      case 'done':
        return 'bg-brand-accent/10 text-brand-accent';
      default:
        return 'bg-brand-elevated text-brand-muted';
    }
  }
}
