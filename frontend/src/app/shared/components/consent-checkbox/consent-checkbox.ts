import { Component, input, model, signal } from '@angular/core';

import { AgreementDoc } from '../../../core/models/legal';
import { AgreementView } from '../agreement/agreement';

/**
 * "I accept the agreement" checkbox. Per the product owner, ticking it at
 * registration / payment is the rental contract, so the full text is one tap away.
 */
@Component({
  selector: 'app-consent-checkbox',
  standalone: true,
  imports: [AgreementView],
  template: `
    <label class="flex cursor-pointer items-start gap-2 text-left">
      <input
        type="checkbox"
        name="acceptedRules"
        [checked]="checked()"
        (change)="checked.set($any($event.target).checked)"
        class="mt-0.5 shrink-0 cursor-pointer rounded border-brand-line text-brand-accent focus:ring-0"
      />
      <span class="text-[11px] leading-relaxed text-brand-muted">
        {{ prefix() }}
        <a href="#" role="button" (click)="$event.preventDefault(); $event.stopPropagation(); open.set(true)" class="font-medium text-brand-accent hover:underline">{{ linkText() }}</a>
      </span>
    </label>

    @if (open()) {
      <div class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4" (click)="open.set(false)">
        <div
          class="flex max-h-[85vh] w-full max-w-lg flex-col rounded-xl border border-brand-line bg-brand-surface p-5 shadow-xl"
          (click)="$event.stopPropagation()"
        >
          <div class="min-h-0 flex-1 overflow-y-auto pr-1 custom-scrollbar">
            <app-agreement [doc]="doc()" />
          </div>
          <div class="mt-4 flex justify-end gap-2 border-t border-brand-line pt-3">
            <button type="button" (click)="open.set(false)" class="rounded-md border border-brand-line px-3 py-2 text-xs text-brand-muted">
              Закрыть
            </button>
            <button
              type="button"
              (click)="checked.set(true); open.set(false)"
              class="rounded-md bg-brand-accent px-4 py-2 text-xs font-semibold text-brand-bg"
            >
              Принимаю
            </button>
          </div>
        </div>
      </div>
    }
  `,
})
export class ConsentCheckbox {
  doc = input<AgreementDoc>('resident_terms');
  prefix = input('Я принимаю');
  linkText = input('Пользовательское соглашение и правила аренды кладовых');
  checked = model(false);
  open = signal(false);
}
