import { Component, inject, input, OnInit, signal } from '@angular/core';
import { DatePipe } from '@angular/common';

import { StorageService } from '../../../../../core/services/storage';

/**
 * Name a successor (e.g. the buyer of the flat) by ИИН, agreement п. 6.2.
 * They may book this storage within 14 days after the current rental ends.
 * Any rule the server enforces (own ИИН, open debt…) comes back as its message.
 */
@Component({
  selector: 'app-transfer-successor',
  standalone: true,
  imports: [DatePipe],
  template: `
    <section class="rounded-2xl border border-brand-line bg-brand-surface px-4 py-4">
      <p class="text-sm font-semibold text-brand-text">Передать кладовку новому владельцу</p>
      <p class="mt-1 text-xs leading-5 text-brand-muted">
        Например, при продаже квартиры. Укажите ИИН нового собственника — после окончания вашей аренды он
        сможет оформить её в течение 14 дней. ОСИ получит уведомление.
      </p>

      @if (currentIIN()) {
        <div class="mt-3 rounded-xl border border-brand-accent/30 bg-brand-accent/5 px-3 py-2.5 text-sm">
          <p class="text-brand-text">
            <i class="fas fa-user-check mr-1 text-brand-accent" aria-hidden="true"></i>
            Новый владелец указан: ИИН <span class="font-semibold tabular-nums">{{ currentIIN() }}</span>
          </p>
          @if (until()) {
            <p class="mt-0.5 text-xs text-brand-muted">Может оформить аренду до {{ until() | date: 'd MMMM y' }}</p>
          }
        </div>
      }

      <div class="mt-3 flex gap-2">
        <input
          type="text"
          inputmode="numeric"
          maxlength="12"
          placeholder="ИИН — 12 цифр"
          aria-label="ИИН нового владельца"
          [value]="iin()"
          (input)="iin.set($any($event.target).value)"
          class="min-w-0 flex-1 rounded-xl border border-brand-line bg-brand-elevated/30 px-3 py-2 text-sm text-brand-text outline-none focus:border-brand-accent"
        />
        <button
          type="button"
          (click)="save()"
          [disabled]="saving()"
          class="h-11 shrink-0 rounded-xl bg-brand-accent px-4 text-sm font-semibold text-brand-bg disabled:opacity-40"
        >
          {{ saving() ? 'Сохранение…' : currentIIN() ? 'Изменить' : 'Указать' }}
        </button>
      </div>
      @if (error()) {
        <p role="alert" class="mt-2 text-sm text-status-overdue">{{ error() }}</p>
      }
      @if (message()) {
        <p role="status" class="mt-2 text-sm text-brand-accent">{{ message() }}</p>
      }
    </section>
  `,
})
export class TransferSuccessor implements OnInit {
  private storages = inject(StorageService);

  storageId = input.required<string>();

  iin = signal('');
  currentIIN = signal('');
  until = signal<string | null>(null);
  saving = signal(false);
  error = signal<string | null>(null);
  message = signal<string | null>(null);

  ngOnInit(): void {
    this.loadCurrent();
  }

  private loadCurrent(): void {
    this.storages.getById(this.storageId()).subscribe({
      next: (s) => {
        this.currentIIN.set(s.transfer_to_iin ?? '');
        this.until.set(s.transfer_until ?? null);
      },
      // Showing the form still works without the current successor.
      error: () => this.currentIIN.set(''),
    });
  }

  save(): void {
    const iin = this.iin().trim();
    if (!/^\d{12}$/.test(iin)) {
      this.error.set('ИИН должен содержать 12 цифр');
      return;
    }
    this.saving.set(true);
    this.error.set(null);
    this.message.set(null);
    this.storages.setTransfer(this.storageId(), iin).subscribe({
      next: (res) => {
        this.saving.set(false);
        this.iin.set('');
        this.message.set(res?.message || 'Новый владелец указан');
        this.loadCurrent();
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err?.error?.error || 'Не удалось указать нового владельца');
      },
    });
  }
}
