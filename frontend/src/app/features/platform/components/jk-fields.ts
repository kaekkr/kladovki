import { Component, model } from '@angular/core';

import { JKForm } from '../../../core/models/platform';

const INPUT =
  'w-full rounded-md border border-brand-line bg-brand-elevated/30 px-3 py-2 text-sm text-brand-text placeholder:text-brand-muted outline-none focus:border-brand-accent';

/** Name, address and cadastral number of a ЖК (two-way bound, updated immutably). */
@Component({
  selector: 'app-jk-fields',
  standalone: true,
  template: `
    <div class="space-y-4">
      <div>
        <label for="jk-name" class="mb-1.5 block text-xs text-brand-muted">Название ЖК *</label>
        <input
          id="jk-name"
          type="text"
          maxlength="255"
          [class]="input"
          [value]="form().name"
          (input)="set('name', $event)"
          placeholder="ЖК «Пример»"
        />
      </div>
      <div>
        <label for="jk-address" class="mb-1.5 block text-xs text-brand-muted">Адрес *</label>
        <input
          id="jk-address"
          type="text"
          maxlength="500"
          [class]="input"
          [value]="form().address"
          (input)="set('address', $event)"
          placeholder="Город, улица, дом"
        />
      </div>
      <div>
        <label for="jk-cadastral" class="mb-1.5 block text-xs text-brand-muted">
          Кадастровый номер ЖК
        </label>
        <input
          id="jk-cadastral"
          type="text"
          inputmode="numeric"
          maxlength="64"
          [class]="input"
          [value]="form().cadastral_number"
          (input)="set('cadastral_number', $event)"
          placeholder="21:319:021:848:8"
        />
        <p class="mt-1.5 text-xs text-brand-muted">
          Необязательно. Номер земельного участка/здания без номера квартиры — по нему жители
          находят свой ЖК при регистрации.
        </p>
      </div>
    </div>
  `,
})
export class JKFields {
  form = model.required<JKForm>();
  readonly input = INPUT;

  set(field: keyof JKForm, event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.form.set({ ...this.form(), [field]: value });
  }
}
