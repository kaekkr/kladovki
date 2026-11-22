import { Injectable, signal } from '@angular/core';

export interface ConfirmRequest {
  title: string;
  message: string;
  confirmLabel: string;
  danger?: boolean;
}

/** In-app replacement for window.confirm (native dialogs look foreign in the Android app). */
@Injectable({ providedIn: 'root' })
export class AdminConfirm {
  readonly current = signal<ConfirmRequest | null>(null);
  private resolve: ((ok: boolean) => void) | null = null;

  ask(req: ConfirmRequest): Promise<boolean> {
    this.answer(false); // a previous dialog left open counts as «Отмена»
    this.current.set(req);
    return new Promise((resolve) => (this.resolve = resolve));
  }

  answer(ok: boolean): void {
    const done = this.resolve;
    this.resolve = null;
    this.current.set(null);
    done?.(ok);
  }
}
