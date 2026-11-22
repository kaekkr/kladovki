import { Component, computed, input, model } from '@angular/core';

import { formatPhone, normalizePhone } from '../../../core/utils/phone';

/**
 * «Номер Kaspi» next to a pay button: prefilled with the profile phone, editable,
 * and says which number the invoice goes to. The parent validates with
 * `phoneProblem()` and sends `normalizePhone()` as `phone`.
 */
@Component({
  selector: 'app-kaspi-phone',
  standalone: true,
  template: `
    <label [for]="inputId()" class="block text-xs font-medium text-brand-muted">Номер Kaspi</label>
    <input
      [id]="inputId()"
      type="tel"
      inputmode="tel"
      autocomplete="tel"
      placeholder="+7 777 000 00 00"
      [value]="phone()"
      (input)="phone.set($any($event.target).value)"
      [attr.aria-invalid]="!!error()"
      class="mt-1 w-full rounded-xl border bg-brand-elevated/30 px-3 py-2.5 text-sm text-brand-text outline-none focus:border-brand-accent"
      [class.border-status-overdue]="!!error()"
      [class.border-brand-line]="!error()"
    />
    @if (error()) {
      <p role="alert" class="mt-1 text-xs text-status-overdue">{{ error() }}</p>
    } @else if (target()) {
      <p class="mt-1 text-xs text-brand-muted">
        Счёт придёт в Kaspi на номер
        <span class="font-semibold text-brand-text">{{ target() }}</span>
      </p>
    } @else {
      <p class="mt-1 text-xs text-brand-subtle">Номер, к которому привязан ваш Kaspi</p>
    }
  `,
})
export class KaspiPhone {
  phone = model('');
  error = input<string | null>(null);
  inputId = input('kaspi-phone');

  target = computed(() => (normalizePhone(this.phone()) ? formatPhone(this.phone()) : ''));
}
