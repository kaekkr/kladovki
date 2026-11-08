import { Component, computed, inject } from '@angular/core';

import { AuthService } from '../../../core/auth/auth.service';
import { ChangePassword } from './change-password';

/** After the ОСИ reset a password, the user must choose their own before continuing. */
@Component({
  selector: 'app-force-password-change',
  standalone: true,
  imports: [ChangePassword],
  template: `
    @if (required()) {
      <div class="fixed inset-0 z-[80] flex items-center justify-center bg-black/70 p-4">
        <div class="w-full max-w-sm space-y-3 rounded-xl border border-brand-line bg-brand-surface p-5">
          <p class="text-sm font-semibold text-brand-text">Придумайте новый пароль</p>
          <p class="text-xs text-brand-muted">
            Вы вошли с временным паролем, который выдал ОСИ. Задайте свой пароль, чтобы продолжить.
          </p>
          <app-change-password currentLabel="Временный пароль" (done)="refresh()" />
        </div>
      </div>
    }
  `,
})
export class ForcePasswordChange {
  private auth = inject(AuthService);
  required = computed(() => !!this.auth.currentUser()?.must_change_password);

  refresh(): void {
    this.auth.getMe().subscribe();
  }
}
