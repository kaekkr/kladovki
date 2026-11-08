import { Component, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AuthService } from '../../../core/auth/auth.service';
import { passwordsProblem } from '../../../core/utils/login-input';

/** Change own password (current one required). Used in profile, settings and the forced-change prompt. */
@Component({
  selector: 'app-change-password',
  standalone: true,
  imports: [FormsModule],
  template: `
    <form class="space-y-2" (ngSubmit)="submit()">
      <input type="password" name="current" autocomplete="current-password" [placeholder]="currentLabel()"
        [ngModel]="current()" (ngModelChange)="current.set($event)" [class]="inputClass" />
      <input type="password" name="next" autocomplete="new-password" placeholder="Новый пароль (минимум 6 символов)"
        [ngModel]="next()" (ngModelChange)="next.set($event)" [class]="inputClass" />
      <input type="password" name="repeat" autocomplete="new-password" placeholder="Повторите новый пароль"
        [ngModel]="repeat()" (ngModelChange)="repeat.set($event)" [class]="inputClass" />
      @if (error()) {
        <p class="text-xs text-status-overdue">{{ error() }}</p>
      }
      @if (saved()) {
        <p class="text-xs text-brand-accent">Пароль изменён</p>
      }
      <button type="submit" [disabled]="saving()"
        class="h-10 w-full rounded-xl bg-brand-accent text-xs font-semibold text-brand-bg disabled:opacity-40">
        {{ saving() ? 'Сохранение…' : 'Сохранить пароль' }}
      </button>
    </form>
  `,
})
export class ChangePassword {
  private auth = inject(AuthService);

  currentLabel = input('Текущий пароль');
  done = output<void>();

  inputClass =
    'w-full rounded-md border border-brand-line bg-brand-elevated/30 px-3 py-2 text-sm text-brand-text outline-none focus:border-brand-accent';
  current = signal('');
  next = signal('');
  repeat = signal('');
  saving = signal(false);
  saved = signal(false);
  error = signal<string | null>(null);

  submit(): void {
    this.saved.set(false);
    const problem = passwordsProblem(this.next(), this.repeat());
    if (!this.current()) {
      this.error.set('Введите текущий пароль');
      return;
    }
    if (problem) {
      this.error.set(problem);
      return;
    }
    this.error.set(null);
    this.saving.set(true);
    this.auth.changePassword(this.current(), this.next()).subscribe({
      next: () => {
        this.saving.set(false);
        this.saved.set(true);
        this.current.set('');
        this.next.set('');
        this.repeat.set('');
        this.done.emit();
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err?.error?.error || 'Не удалось изменить пароль');
      },
    });
  }
}
