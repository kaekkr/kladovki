import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';

import { AuthService } from '../../../../core/auth/auth.service';
import { User, UserStatus } from '../../../../core/auth/auth.models';
import { ResidentService, ReviewAction } from '../../../../core/services/resident';
import { propertyLabel, residentStatusLabel } from '../../../../core/utils/resident-display';

type Filter = UserStatus | 'all';

/** ОСИ confirms that new users own a flat or premises in the ЖК (user agreement п. 2.1). */
@Component({
  selector: 'app-admin-residents',
  standalone: true,
  imports: [DatePipe, FormsModule],
  templateUrl: './residents.html',
})
export class AdminResidents {
  private auth = inject(AuthService);
  private residents = inject(ResidentService);

  list = signal<User[]>([]);
  loading = signal(true);
  error = signal(false);
  filter = signal<Filter>('pending');

  // Decision dialog (reject / block need a comment)
  active = signal<User | null>(null);
  action = signal<ReviewAction>('reject');
  comment = signal('');
  saving = signal(false);
  actionError = signal<string | null>(null);

  statusLabel = residentStatusLabel;
  filters: { key: Filter; label: string }[] = [
    { key: 'pending', label: 'На проверке' },
    { key: 'active', label: 'Подтверждены' },
    { key: 'rejected', label: 'Отклонены' },
    { key: 'blocked', label: 'Заблокированы' },
    { key: 'all', label: 'Все' },
  ];

  pendingCount = computed(() => this.list().filter((u) => u.status === 'pending').length);
  visible = computed(() => {
    const f = this.filter();
    return f === 'all' ? this.list() : this.list().filter((u) => u.status === f);
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
    this.residents.listByJK(jkId).subscribe({
      next: (list) => {
        this.list.set(list);
        this.loading.set(false);
        this.error.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }

  property(u: User): string {
    return propertyLabel({
      owner_type: u.owner_type ?? 'person',
      premises_kind: u.premises_kind ?? 'flat',
      premises_number: u.premises_number || '—',
      company_bin: u.company_bin,
      company_name: u.company_name,
    });
  }

  approve(u: User): void {
    this.send(u, 'approve', '');
  }

  open(u: User, action: ReviewAction): void {
    this.active.set(u);
    this.action.set(action);
    this.comment.set('');
    this.actionError.set(null);
  }

  submit(): void {
    const u = this.active();
    if (!u) return;
    if (this.action() === 'reject' && !this.comment().trim()) {
      this.actionError.set('Укажите причину — житель увидит её');
      return;
    }
    this.send(u, this.action(), this.comment().trim());
  }

  private send(u: User, action: ReviewAction, comment: string): void {
    this.saving.set(true);
    this.actionError.set(null);
    this.residents.review(u.id, action, comment).subscribe({
      next: () => {
        this.saving.set(false);
        this.active.set(null);
        this.load();
      },
      error: (err) => {
        this.saving.set(false);
        this.actionError.set(err?.error?.error || 'Не удалось сохранить решение');
      },
    });
  }

  statusClass(status: UserStatus | undefined): string {
    switch (status) {
      case 'pending':
        return 'bg-orange-500/10 text-orange-500';
      case 'active':
        return 'bg-brand-accent/10 text-brand-accent';
      case 'rejected':
      case 'blocked':
        return 'bg-red-500/10 text-red-500';
      default:
        return 'bg-brand-elevated text-brand-muted';
    }
  }
}
