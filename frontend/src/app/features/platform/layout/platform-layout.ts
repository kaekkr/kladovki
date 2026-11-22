import { Component, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterOutlet } from '@angular/router';

import { AuthService } from '../../../core/auth/auth.service';
import { ForcePasswordChange } from '../../../shared/components/change-password/force-password-change';

/** Shell of the super-admin panel: a top bar with the operator's name and logout. */
@Component({
  selector: 'app-platform-layout',
  standalone: true,
  imports: [RouterOutlet, RouterLink, ForcePasswordChange],
  template: `
    <div class="min-h-screen bg-brand-bg text-brand-text font-sans antialiased">
      <header class="sticky top-0 z-10 border-b border-brand-line bg-brand-bg">
        <div class="mx-auto flex h-14 max-w-5xl items-center justify-between gap-3 px-4 sm:px-6">
          <a routerLink="/platform" class="min-w-0">
            <p class="truncate text-[13px] font-medium text-brand-text">Кладовки — платформа</p>
            <p class="truncate text-[11px] text-brand-subtle">
              {{ user()?.full_name || 'Оператор платформы' }}
            </p>
          </a>
          <button
            type="button"
            (click)="logout()"
            [disabled]="leaving()"
            class="flex shrink-0 items-center gap-2 rounded-md border border-brand-line px-3 py-1.5 text-xs font-medium text-brand-muted hover:border-brand-accent hover:text-brand-text disabled:opacity-50"
          >
            <i class="fas fa-right-from-bracket text-[11px]"></i>
            Выйти
          </button>
        </div>
      </header>

      <main class="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6">
        <router-outlet />
      </main>
    </div>

    <app-force-password-change />
  `,
})
export class PlatformLayout {
  private auth = inject(AuthService);
  private router = inject(Router);

  user = this.auth.currentUser;
  leaving = signal(false);

  logout(): void {
    this.leaving.set(true);
    this.auth.logout().subscribe(() => this.router.navigate(['/login']));
  }
}
