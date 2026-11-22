import { Component, model } from '@angular/core';

import { AccountForm } from '../../../core/models/platform';

const INPUT =
  'w-full rounded-md border border-brand-line bg-brand-elevated/30 px-3 py-2 text-sm text-brand-text placeholder:text-brand-muted outline-none focus:border-brand-accent';

/** Full name, email and phone of an ОСИ account (two-way bound, updated immutably). */
@Component({
  selector: 'app-account-fields',
  standalone: true,
  template: `
    <div class="space-y-4">
      <div>
        <label [for]="id('name')" class="mb-1.5 block text-xs text-brand-muted">ФИО *</label>
        <input
          [id]="id('name')"
          type="text"
          maxlength="255"
          autocomplete="off"
          [class]="input"
          [value]="form().full_name"
          (input)="set('full_name', $event)"
          placeholder="Иванов Иван"
        />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label [for]="id('email')" class="mb-1.5 block text-xs text-brand-muted"
            >Email (логин) *</label
          >
          <input
            [id]="id('email')"
            type="email"
            inputmode="email"
            maxlength="255"
            autocomplete="off"
            [class]="input"
            [value]="form().email"
            (input)="set('email', $event)"
            placeholder="osi@example.kz"
          />
        </div>
        <div>
          <label [for]="id('phone')" class="mb-1.5 block text-xs text-brand-muted">Телефон *</label>
          <input
            [id]="id('phone')"
            type="tel"
            inputmode="tel"
            maxlength="50"
            autocomplete="off"
            [class]="input"
            [value]="form().phone"
            (input)="set('phone', $event)"
            placeholder="+7 701 123 45 67"
          />
        </div>
      </div>
    </div>
  `,
})
export class AccountFields {
  form = model.required<AccountForm>();
  readonly input = INPUT;
  private static seq = 0;
  private readonly prefix = `acc-${AccountFields.seq++}`;

  id(field: string): string {
    return `${this.prefix}-${field}`;
  }

  set(field: keyof AccountForm, event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.form.set({ ...this.form(), [field]: value });
  }
}
