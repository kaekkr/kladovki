import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { LegalService } from '../../../../core/services/legal';
import { AgreementView } from '../../../../shared/components/agreement/agreement';

/** Asks the ОСИ to accept the current «Регламент для ОСИ» before using the admin panel. */
@Component({
  selector: 'app-osi-rules-prompt',
  standalone: true,
  imports: [AgreementView],
  template: `
    @if (version()) {
      <div class="fixed inset-0 z-[70] flex items-center justify-center bg-black/70 p-4">
        <div class="flex max-h-[90vh] w-full max-w-2xl flex-col rounded-xl border border-brand-line bg-brand-surface p-6 shadow-xl">
          <p class="text-xs font-medium uppercase tracking-wide text-brand-accent">Перед началом работы</p>
          <p class="mt-1 text-sm text-brand-muted">
            Ознакомьтесь с регламентом использования сервиса для ОСИ и подтвердите согласие.
          </p>
          <div class="mt-4 min-h-0 flex-1 overflow-y-auto rounded-lg border border-brand-line p-4 custom-scrollbar">
            <app-agreement doc="osi_rules" />
          </div>
          @if (error()) {
            <p class="mt-3 text-xs text-status-overdue">{{ error() }}</p>
          }
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" (click)="logout()" class="rounded-md border border-brand-line px-4 py-2 text-xs text-brand-muted">
              Выйти
            </button>
            <button
              type="button"
              (click)="accept()"
              [disabled]="saving()"
              class="rounded-md bg-brand-accent px-4 py-2 text-xs font-semibold text-brand-bg disabled:opacity-40"
            >
              Принимаю регламент
            </button>
          </div>
        </div>
      </div>
    }
  `,
})
export class OsiRulesPrompt {
  private legal = inject(LegalService);
  private auth = inject(AuthService);
  private router = inject(Router);

  /** Set while the current version is not yet accepted. */
  version = signal('');
  saving = signal(false);
  error = signal<string | null>(null);

  constructor() {
    this.legal.status().subscribe({
      next: (s) => this.version.set(s.osi_rules.accepted ? '' : s.osi_rules.version),
      error: () => this.version.set(''), // never lock the panel because of a network error
    });
  }

  accept(): void {
    this.saving.set(true);
    this.legal.accept('osi_rules', this.version()).subscribe({
      next: () => {
        this.saving.set(false);
        this.version.set('');
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err?.error?.error || 'Не удалось сохранить согласие');
      },
    });
  }

  logout(): void {
    this.auth.logout().subscribe(() => this.router.navigate(['/login']));
  }
}
