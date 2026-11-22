import { Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { AccountForm, CreateJKResponse, JKForm } from '../../../core/models/platform';
import { PlatformService } from '../../../core/services/platform';
import {
  apiErrorText,
  validateAccountForm,
  validateJKForm,
} from '../../../core/utils/platform-form';
import { AccountFields } from '../components/account-fields';
import { JKFields } from '../components/jk-fields';
import { TempPassword } from '../components/temp-password';

const EMPTY_JK: JKForm = { name: '', address: '', cadastral_number: '' };
const EMPTY_ACCOUNT: AccountForm = { full_name: '', email: '', phone: '' };

/** Creates a ЖК together with its first ОСИ account; shows the temporary password once. */
@Component({
  selector: 'app-platform-jk-create',
  standalone: true,
  imports: [RouterLink, JKFields, AccountFields, TempPassword],
  template: `
    <div class="max-w-2xl space-y-6">
      <div>
        <a routerLink="/platform" class="text-xs text-brand-muted hover:text-brand-text"
          >← Все ЖК</a
        >
        <h1 class="mt-2 text-2xl font-semibold tracking-tight text-brand-text">Новый ЖК</h1>
        <p class="mt-1 text-sm text-brand-muted">ЖК и учётная запись его администратора ОСИ</p>
      </div>

      @if (created(); as res) {
        <div class="space-y-4">
          <div
            class="rounded-lg border border-brand-line bg-brand-surface p-4 text-sm text-brand-text"
          >
            ЖК «{{ res.jk.name }}» создан. Администратор ОСИ: {{ res.admin.full_name }}.
          </div>
          @if (!passwordSaved()) {
            <app-temp-password
              title="Временный пароль администратора ОСИ"
              [login]="res.admin.email"
              [password]="res.temporary_password"
              (closed)="passwordSaved.set(true)"
            />
          }
          <div class="flex flex-wrap gap-2">
            <a
              [routerLink]="['/platform/jks', res.jk.id]"
              class="rounded-md bg-brand-accent px-4 py-2 text-sm font-semibold text-brand-bg hover:opacity-90"
            >
              Открыть ЖК
            </a>
            <a
              routerLink="/platform"
              class="rounded-md border border-brand-line px-4 py-2 text-sm text-brand-text hover:border-brand-accent"
            >
              К списку
            </a>
          </div>
        </div>
      } @else {
        <form class="space-y-6" (submit)="$event.preventDefault(); submit()" novalidate>
          <section class="overflow-hidden rounded-lg border border-brand-line bg-brand-surface">
            <div class="border-b border-brand-line px-5 py-4">
              <h2 class="text-sm font-medium text-brand-text">Жилой комплекс</h2>
            </div>
            <div class="px-5 py-4">
              <app-jk-fields [(form)]="jk" />
            </div>
          </section>

          <section class="overflow-hidden rounded-lg border border-brand-line bg-brand-surface">
            <div class="border-b border-brand-line px-5 py-4">
              <h2 class="text-sm font-medium text-brand-text">Администратор ОСИ</h2>
              <p class="mt-0.5 text-xs text-brand-muted">
                Пароль сгенерируется автоматически и будет показан один раз
              </p>
            </div>
            <div class="px-5 py-4">
              <app-account-fields [(form)]="admin" />
            </div>
          </section>

          @if (error()) {
            <div
              class="rounded-lg border border-brand-line bg-brand-surface px-4 py-3 text-sm text-status-overdue"
              role="alert"
            >
              {{ error() }}
            </div>
          }

          <button
            type="submit"
            [disabled]="saving()"
            class="w-full rounded-md bg-brand-accent px-4 py-2.5 text-sm font-semibold text-brand-bg hover:opacity-90 disabled:opacity-50 sm:w-auto"
          >
            {{ saving() ? 'Создаём…' : 'Создать ЖК и учётную запись ОСИ' }}
          </button>
        </form>
      }
    </div>
  `,
})
export class PlatformJKCreate {
  private platform = inject(PlatformService);

  jk = signal<JKForm>(EMPTY_JK);
  admin = signal<AccountForm>(EMPTY_ACCOUNT);
  saving = signal(false);
  error = signal<string | null>(null);
  created = signal<CreateJKResponse | null>(null);
  passwordSaved = signal(false);

  submit(): void {
    const problem = validateJKForm(this.jk()) ?? validateAccountForm(this.admin());
    if (problem) {
      this.error.set(problem);
      return;
    }
    this.saving.set(true);
    this.error.set(null);
    this.platform.createJK({ ...this.jk(), admin: this.admin() }).subscribe({
      next: (res) => {
        this.saving.set(false);
        this.created.set(res);
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(apiErrorText(err, 'Не удалось создать ЖК. Попробуйте ещё раз.'));
      },
    });
  }
}
