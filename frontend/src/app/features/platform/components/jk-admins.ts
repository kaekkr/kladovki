import { Component, inject, input, output, signal } from '@angular/core';
import { Observable } from 'rxjs';

import { AccountForm, PlatformAdmin } from '../../../core/models/platform';
import { PlatformService } from '../../../core/services/platform';
import { apiErrorText, validateAccountForm } from '../../../core/utils/platform-form';
import { AccountFields } from './account-fields';
import { TempPassword } from './temp-password';

const EMPTY_ACCOUNT: AccountForm = { full_name: '', email: '', phone: '' };

interface IssuedPassword {
  title: string;
  login: string;
  password: string;
}

/** ОСИ accounts of one ЖК: list, add another, reset a password. */
@Component({
  selector: 'app-jk-admins',
  standalone: true,
  imports: [AccountFields, TempPassword],
  template: `
    <section class="overflow-hidden rounded-lg border border-brand-line bg-brand-surface">
      <div
        class="flex flex-wrap items-center justify-between gap-2 border-b border-brand-line px-5 py-4"
      >
        <div>
          <h2 class="text-sm font-medium text-brand-text">Администраторы ОСИ</h2>
          <p class="mt-0.5 text-xs text-brand-muted">Входят по email и паролю</p>
        </div>
        @if (!adding()) {
          <button
            type="button"
            (click)="startAdding()"
            class="rounded-md border border-brand-line px-3 py-1.5 text-xs font-medium text-brand-text hover:border-brand-accent"
          >
            + Добавить ОСИ
          </button>
        }
      </div>

      <div class="space-y-4 px-5 py-4">
        @if (issued(); as p) {
          <app-temp-password
            [title]="p.title"
            [login]="p.login"
            [password]="p.password"
            (closed)="issued.set(null)"
          />
        }
        @if (error()) {
          <div
            class="rounded-md border border-brand-line px-3 py-2 text-sm text-status-overdue"
            role="alert"
          >
            {{ error() }}
          </div>
        }

        @if (adding()) {
          <form
            class="space-y-4 rounded-lg border border-brand-line p-4"
            (submit)="$event.preventDefault(); add()"
            novalidate
          >
            <app-account-fields [(form)]="draft" />
            <div class="flex flex-wrap gap-2">
              <button
                type="submit"
                [disabled]="busy()"
                class="rounded-md bg-brand-accent px-4 py-2 text-xs font-semibold text-brand-bg hover:opacity-90 disabled:opacity-50"
              >
                {{ busy() ? 'Создаём…' : 'Создать учётную запись' }}
              </button>
              <button
                type="button"
                (click)="adding.set(false)"
                class="rounded-md border border-brand-line px-4 py-2 text-xs text-brand-text hover:border-brand-accent"
              >
                Отмена
              </button>
            </div>
          </form>
        }

        @if (admins().length === 0) {
          <p class="text-sm text-brand-muted">У этого ЖК пока нет администраторов.</p>
        }
        <ul class="divide-y divide-brand-line">
          @for (a of admins(); track a.id) {
            <li class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <p class="text-sm font-medium text-brand-text">{{ a.full_name }}</p>
                <p class="text-xs text-brand-muted break-all">{{ a.email }} · {{ a.phone }}</p>
                @if (a.must_change_password) {
                  <p class="mt-0.5 text-xs text-orange-500">Ещё не сменил временный пароль</p>
                }
              </div>
              <button
                type="button"
                (click)="reset(a)"
                [disabled]="busy()"
                class="shrink-0 self-start rounded-md border border-brand-line px-3 py-1.5 text-xs font-medium text-brand-text hover:border-brand-accent disabled:opacity-50 sm:self-auto"
              >
                Сбросить пароль
              </button>
            </li>
          }
        </ul>
      </div>
    </section>
  `,
})
export class JKAdmins {
  private platform = inject(PlatformService);

  jkId = input.required<string>();
  admins = input.required<PlatformAdmin[]>();
  /** Asks the page to reload the list after a change. */
  changed = output<void>();

  adding = signal(false);
  draft = signal<AccountForm>(EMPTY_ACCOUNT);
  busy = signal(false);
  error = signal<string | null>(null);
  issued = signal<IssuedPassword | null>(null);

  startAdding(): void {
    this.draft.set(EMPTY_ACCOUNT);
    this.error.set(null);
    this.adding.set(true);
  }

  add(): void {
    const problem = validateAccountForm(this.draft());
    if (problem) {
      this.error.set(problem);
      return;
    }
    this.run(
      this.platform.addAdmin(this.jkId(), this.draft()),
      'Не удалось добавить администратора.',
      (res) => {
        this.adding.set(false);
        this.issued.set({
          title: `Временный пароль для ${res.admin.full_name}`,
          login: res.admin.email,
          password: res.temporary_password,
        });
      },
    );
  }

  reset(a: PlatformAdmin): void {
    if (!confirm(`Сбросить пароль для ${a.full_name}? Старый пароль перестанет работать.`)) return;
    this.run(this.platform.resetAdminPassword(a.id), 'Не удалось сбросить пароль.', (res) => {
      this.issued.set({
        title: `Новый временный пароль для ${a.full_name}`,
        login: a.email,
        password: res.temporary_password,
      });
    });
  }

  private run<T>(
    req: Observable<T>,
    fallback: string,
    done: (res: T) => void,
  ): void {
    this.busy.set(true);
    this.error.set(null);
    this.issued.set(null);
    req.subscribe({
      next: (res) => {
        this.busy.set(false);
        done(res);
        this.changed.emit();
      },
      error: (err) => {
        this.busy.set(false);
        this.error.set(apiErrorText(err, fallback));
      },
    });
  }
}
