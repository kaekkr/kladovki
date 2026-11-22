import { Component, inject, input, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { FormsModule } from '@angular/forms';

import { formatPhone, normalizePhone, phoneProblem } from '../../../../core/utils/phone';
import { ProfilePhoneService } from './profile-phone.service';

/** Profile row «Телефон» with «Изменить»: the number Kaspi invoices go to by default. */
@Component({
  selector: 'app-phone-editor',
  standalone: true,
  imports: [FormsModule],
  template: `
    <div class="flex items-center gap-3 px-4 py-4">
      <div
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-brand-elevated text-brand-muted"
      >
        <i class="fas fa-phone text-xs" aria-hidden="true"></i>
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-xs text-brand-subtle">Телефон (на него приходит счёт Kaspi)</p>
        <p class="mt-0.5 truncate text-sm font-medium text-brand-text">
          {{ phone() ? display(phone()) : 'Не указан' }}
        </p>
      </div>
      @if (!editing()) {
        <button
          type="button"
          (click)="open()"
          class="shrink-0 text-sm font-semibold text-brand-accent"
        >
          Изменить
        </button>
      }
    </div>
    @if (saved()) {
      <p role="status" class="px-4 pb-3 text-xs text-brand-accent">Номер сохранён</p>
    }
    @if (editing()) {
      <form class="space-y-2 px-4 pb-4" (ngSubmit)="save()">
        <label for="profile-phone" class="block text-xs text-brand-muted">Новый номер</label>
        <input
          id="profile-phone"
          name="phone"
          type="tel"
          inputmode="tel"
          autocomplete="tel"
          placeholder="+7 777 000 00 00"
          [ngModel]="draft()"
          (ngModelChange)="draft.set($event)"
          class="w-full rounded-md border border-brand-line bg-brand-elevated/30 px-3 py-2 text-sm text-brand-text outline-none focus:border-brand-accent"
        />
        @if (error()) {
          <p role="alert" class="text-xs text-status-overdue">{{ error() }}</p>
        }
        <div class="flex gap-2">
          <button
            type="button"
            (click)="editing.set(false)"
            class="h-10 flex-1 rounded-xl border border-brand-line text-sm text-brand-muted"
          >
            Отмена
          </button>
          <button
            type="submit"
            [disabled]="saving()"
            class="h-10 flex-1 rounded-xl bg-brand-accent text-sm font-semibold text-brand-bg disabled:opacity-40"
          >
            {{ saving() ? 'Сохранение…' : 'Сохранить' }}
          </button>
        </div>
      </form>
    }
  `,
})
export class PhoneEditor {
  private service = inject(ProfilePhoneService);

  phone = input<string>('');

  editing = signal(false);
  draft = signal('');
  saving = signal(false);
  saved = signal(false);
  error = signal<string | null>(null);
  display = formatPhone;

  open(): void {
    this.draft.set(this.phone() ? formatPhone(this.phone()) : '');
    this.error.set(null);
    this.saved.set(false);
    this.editing.set(true);
  }

  save(): void {
    const problem = phoneProblem(this.draft());
    const phone = normalizePhone(this.draft());
    if (problem || !phone) {
      this.error.set(problem);
      return;
    }
    this.saving.set(true);
    this.error.set(null);
    this.service.updatePhone(phone).subscribe({
      next: () => {
        this.saving.set(false);
        this.editing.set(false);
        this.saved.set(true);
      },
      error: (err: HttpErrorResponse) => {
        this.saving.set(false);
        this.error.set(
          err.status === 404 || err.status === 405
            ? 'Изменить номер в приложении пока нельзя — обратитесь в ОСИ. Номер для оплаты можно указать на экране оплаты.'
            : err.error?.error || 'Не удалось сохранить номер',
        );
      },
    });
  }
}
