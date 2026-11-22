import { Component, HostListener, inject } from '@angular/core';

import { AdminConfirm } from './confirm';

@Component({
  selector: 'app-admin-confirm-dialog',
  standalone: true,
  template: `
    @if (confirm.current(); as c) {
      <div class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="admin-confirm-title"
          class="w-full max-w-md space-y-4 rounded-xl border border-brand-line bg-brand-surface p-6"
        >
          <h2 id="admin-confirm-title" class="text-base font-semibold text-brand-text">{{ c.title }}</h2>
          <p class="text-sm leading-6 text-brand-muted">{{ c.message }}</p>
          <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <button
              type="button"
              (click)="confirm.answer(false)"
              class="rounded-md border border-brand-line px-4 py-2.5 text-sm font-medium text-brand-text"
            >
              Отмена
            </button>
            <button
              type="button"
              (click)="confirm.answer(true)"
              class="rounded-md px-4 py-2.5 text-sm font-semibold"
              [class]="c.danger ? 'bg-status-overdue text-white' : 'bg-brand-accent text-brand-bg'"
            >
              {{ c.confirmLabel }}
            </button>
          </div>
        </div>
      </div>
    }
  `,
})
export class AdminConfirmDialog {
  confirm = inject(AdminConfirm);

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.confirm.current()) this.confirm.answer(false);
  }
}
