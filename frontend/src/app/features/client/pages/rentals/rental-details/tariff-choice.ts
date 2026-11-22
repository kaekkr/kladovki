import { Component, computed, input, output, signal } from '@angular/core';
import { DatePipe, DecimalPipe } from '@angular/common';

import { TenantRequest } from '../../../../../core/models/request';
import { tariffExtendEndsAt } from '../../../../../core/utils/rental-money';
import { ConfirmDialog } from '../../../../../shared/components/confirm-dialog/confirm-dialog';

type Choice = 'extend' | 'refund';

/**
 * After a tariff decrease the tenant picks: a longer free period or money back.
 * Both are irreversible, so the result is shown up front and confirmed in a dialog.
 */
@Component({
  selector: 'app-tariff-choice',
  standalone: true,
  imports: [ConfirmDialog, DatePipe, DecimalPipe],
  template: `
    <section class="rounded-2xl border border-status-locked/30 bg-status-locked/5 px-4 py-4">
      <p class="text-sm font-semibold text-brand-text">Тариф снижен — выберите, что вам удобнее</p>
      <p class="mt-1 text-xs leading-5 text-brand-muted">
        Цена кладовки снизилась с {{ request().old_price | number: '1.0-0' }} ₸ до
        {{ request().new_price | number: '1.0-0' }} ₸ в месяц. Выберите один вариант — изменить выбор потом
        нельзя.
      </p>
      <div class="mt-3 grid grid-cols-1 gap-2">
        <button
          type="button"
          [disabled]="busy()"
          (click)="pending.set('extend')"
          class="min-h-12 rounded-xl border border-brand-accent bg-brand-accent/10 px-3 py-2 text-sm font-semibold text-brand-accent disabled:opacity-40"
        >
          Продлить срок бесплатно — до {{ newEndsAt() | date: 'd MMMM y' }}
        </button>
        <button
          type="button"
          [disabled]="busy()"
          (click)="pending.set('refund')"
          class="min-h-12 rounded-xl border border-brand-line px-3 py-2 text-sm font-semibold text-brand-text disabled:opacity-40"
        >
          Вернуть {{ request().amount | number: '1.0-0' }} ₸
        </button>
      </div>
      <p class="mt-2 text-xs leading-5 text-brand-subtle">
        Сейчас аренда до {{ endsAt() | date: 'd MMMM y' }}. Деньги возвращает ОСИ самостоятельно (на счёт,
        наличными и т.д.).
      </p>
    </section>

    @if (pending(); as p) {
      <app-confirm-dialog
        [title]="p === 'extend' ? 'Продлить срок бесплатно?' : 'Вернуть деньги?'"
        [confirmLabel]="p === 'extend' ? 'Да, продлить' : 'Да, вернуть'"
        [busy]="busy()"
        (confirmed)="confirm(p)"
        (cancelled)="pending.set(null)"
      >
        @if (p === 'extend') {
          Аренда продлится до <b class="text-brand-text">{{ newEndsAt() | date: 'd MMMM y' }}</b> без оплаты.
          Деньги при этом не возвращаются.
        } @else {
          ОСИ вернёт вам <b class="text-brand-text">{{ request().amount | number: '1.0-0' }} ₸</b>. Срок аренды
          не изменится — до {{ endsAt() | date: 'd MMMM y' }}.
        }
      </app-confirm-dialog>
    }
  `,
})
export class TariffChoice {
  request = input.required<TenantRequest>();
  endsAt = input.required<string>();
  busy = input(false);
  choose = output<Choice>();

  pending = signal<Choice | null>(null);

  newEndsAt = computed(() =>
    tariffExtendEndsAt(this.endsAt(), this.request().old_price, this.request().new_price),
  );

  confirm(choice: Choice): void {
    this.pending.set(null);
    this.choose.emit(choice);
  }
}
