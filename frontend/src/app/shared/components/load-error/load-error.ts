import { Component, input, output } from '@angular/core';

/**
 * «Не удалось загрузить …» with a «Повторить» button. Every page that loads
 * data uses it, so a timeout or a sleeping server never leaves a dead screen.
 */
@Component({
  selector: 'app-load-error',
  standalone: true,
  template: `
    <div
      role="alert"
      class="flex flex-wrap items-center gap-3 rounded-lg border border-brand-line bg-brand-surface p-5 text-sm text-status-overdue"
    >
      <span>{{ message() }}</span>
      <button
        type="button"
        (click)="retry.emit()"
        class="rounded-md border border-brand-line px-3 py-1.5 text-xs font-medium text-brand-text hover:border-brand-accent"
      >
        Повторить
      </button>
    </div>
  `,
})
export class LoadError {
  message = input('Не удалось загрузить данные.');
  retry = output<void>();
}
