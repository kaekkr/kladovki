import { Component, input, output, signal } from '@angular/core';

/** Shows a generated temporary password once, with a copy button and a warning. */
@Component({
  selector: 'app-temp-password',
  standalone: true,
  template: `
    <div class="space-y-3 rounded-lg border border-orange-500/40 bg-orange-500/5 p-4" role="alert">
      <p class="text-sm font-medium text-brand-text">{{ title() }}</p>
      @if (login()) {
        <p class="text-xs text-brand-muted">
          Логин: <span class="font-medium text-brand-text break-all">{{ login() }}</span>
        </p>
      }
      <div class="flex flex-col gap-2 sm:flex-row">
        <input
          type="text"
          readonly
          [value]="password()"
          aria-label="Временный пароль"
          class="w-full rounded-md border border-brand-line bg-brand-elevated/30 px-3 py-2 font-mono text-base tracking-wider text-brand-text outline-none"
        />
        <button
          type="button"
          (click)="copy()"
          class="shrink-0 rounded-md bg-brand-accent px-4 py-2 text-xs font-semibold text-brand-bg hover:opacity-90"
        >
          {{ copied() ? 'Скопировано!' : copyFailed() ? 'Выделите и скопируйте' : 'Копировать' }}
        </button>
      </div>
      <p class="text-xs text-orange-500">
        Пароль показывается только один раз. Сохраните его и передайте лично. При первом входе
        система попросит придумать новый пароль.
      </p>
      <button
        type="button"
        (click)="closed.emit()"
        class="rounded-md border border-brand-line px-3 py-1.5 text-xs font-medium text-brand-text hover:border-brand-accent"
      >
        Я сохранил пароль
      </button>
    </div>
  `,
})
export class TempPassword {
  title = input('Временный пароль');
  login = input<string>('');
  password = input.required<string>();
  closed = output<void>();

  copied = signal(false);
  copyFailed = signal(false);

  copy(): void {
    const clipboard = navigator.clipboard;
    if (!clipboard) {
      this.copyFailed.set(true);
      return;
    }
    clipboard.writeText(this.password()).then(
      () => this.copied.set(true),
      () => this.copyFailed.set(true),
    );
  }
}
