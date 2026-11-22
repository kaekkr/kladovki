import { Component, ElementRef, afterNextRender, inject, input, output } from '@angular/core';

/**
 * In-app confirmation (instead of window.confirm, which looks broken in the
 * Android app). The body is projected: `<app-confirm-dialog …>text</app-confirm-dialog>`.
 */
@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  host: { '(document:keydown.escape)': 'cancelled.emit()' },
  template: `
    <div class="fixed inset-0 z-[60] flex items-end justify-center bg-black/60 p-4 sm:items-center">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        class="w-full max-w-sm rounded-2xl border border-brand-line bg-brand-surface p-5 shadow-2xl"
      >
        <h2 id="confirm-dialog-title" class="text-base font-semibold text-brand-text">
          {{ title() }}
        </h2>
        <div class="mt-2 text-sm leading-6 text-brand-muted"><ng-content /></div>
        <div class="mt-5 grid grid-cols-2 gap-2">
          <button
            type="button"
            data-autofocus
            (click)="cancelled.emit()"
            class="h-12 rounded-xl border border-brand-line text-sm font-medium text-brand-text"
          >
            {{ cancelLabel() }}
          </button>
          <button
            type="button"
            [disabled]="busy()"
            (click)="confirmed.emit()"
            class="h-12 rounded-xl bg-brand-accent text-sm font-semibold text-brand-bg disabled:opacity-40"
          >
            {{ busy() ? 'Сохранение…' : confirmLabel() }}
          </button>
        </div>
      </div>
    </div>
  `,
})
export class ConfirmDialog {
  title = input.required<string>();
  confirmLabel = input('Подтвердить');
  cancelLabel = input('Отмена');
  busy = input(false);
  confirmed = output<void>();
  cancelled = output<void>();

  constructor() {
    const host = inject(ElementRef<HTMLElement>);
    // Focus the safe option, so an accidental «Enter» does not confirm.
    afterNextRender(() => host.nativeElement.querySelector('[data-autofocus]')?.focus());
  }
}
