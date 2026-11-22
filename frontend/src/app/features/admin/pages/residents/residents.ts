import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';

import { AuthService } from '../../../../core/auth/auth.service';
import { User, UserStatus } from '../../../../core/auth/auth.models';
import { ResidentService, ReviewAction } from '../../../../core/services/resident';
import { propertyLabel, residentStatusLabel } from '../../../../core/utils/resident-display';
import { matchesSearch } from '../../../../core/utils/admin-search';
import { LoadError } from '../../../../shared/components/load-error/load-error';
import { AdminConfirm } from '../../components/confirm-dialog/confirm';
import { AdminBadges } from '../../layout/admin-badges';

type Filter = UserStatus | 'all';

/** ОСИ confirms that new users own a flat or premises in the ЖК (user agreement п. 2.1). */
@Component({
  selector: 'app-admin-residents',
  standalone: true,
  imports: [DatePipe, FormsModule, LoadError],
  templateUrl: './residents.html',
})
export class AdminResidents {
  private auth = inject(AuthService);
  private residents = inject(ResidentService);
  private confirm = inject(AdminConfirm);
  private badges = inject(AdminBadges);

  list = signal<User[]>([]);
  loading = signal(true);
  error = signal(false);
  filter = signal<Filter>('pending');
  search = signal('');

  // Decision dialog (reject / block need a comment)
  active = signal<User | null>(null);
  action = signal<ReviewAction>('reject');
  comment = signal('');
  saving = signal(false);
  actionError = signal<string | null>(null);

  statusLabel = residentStatusLabel;

  // Password reset: the temporary password is shown once to the ОСИ.
  resetFor = signal<User | null>(null);
  tempPassword = signal<string | null>(null);
  resetError = signal<string | null>(null);
  copied = signal(false);

  async resetPassword(u: User): Promise<void> {
    const ok = await this.confirm.ask({
      title: 'Сбросить пароль',
      message: `Житель ${u.full_name} получит временный пароль, старый перестанет работать. При входе он должен будет придумать новый.`,
      confirmLabel: 'Сбросить пароль',
    });
    if (!ok) return;
    this.resetFor.set(u);
    this.tempPassword.set(null);
    this.resetError.set(null);
    this.copied.set(false);
    this.residents.resetPassword(u.id).subscribe({
      next: (r) => this.tempPassword.set(r.temporary_password),
      error: (err) => this.resetError.set(err?.error?.error || 'Не удалось сбросить пароль'),
    });
  }

  copyTemp(): void {
    const p = this.tempPassword();
    if (!p) return;
    navigator.clipboard?.writeText(p);
    this.copied.set(true);
  }
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
    const q = this.search();
    return this.list().filter(
      (u) =>
        (f === 'all' || u.status === f) &&
        matchesSearch([u.full_name, u.phone, u.iin, u.premises_number, u.premises_cadastral, u.company_name], q),
    );
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
        this.badges.refresh();
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
