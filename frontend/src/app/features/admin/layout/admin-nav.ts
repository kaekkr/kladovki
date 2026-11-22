import { Injectable, inject, signal } from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { filter } from 'rxjs';

/**
 * Phones and small tablets: the admin menu is a slide-out drawer (☰ in the
 * header) instead of a fixed sidebar that left 158 px for the content.
 */
@Injectable({ providedIn: 'root' })
export class AdminNav {
  readonly open = signal(false);
  /** lg breakpoint (1024 px): the menu is a fixed sidebar, not a drawer. */
  readonly wide = signal(typeof matchMedia === 'function' && matchMedia('(min-width: 1024px)').matches);

  constructor() {
    if (typeof matchMedia === 'function') {
      matchMedia('(min-width: 1024px)').addEventListener('change', (e) => this.wide.set(e.matches));
    }
    inject(Router)
      .events.pipe(filter((e) => e instanceof NavigationEnd))
      .subscribe(() => this.open.set(false));
  }

  /** A closed drawer must not be reachable by keyboard or screen reader. */
  hidden(): boolean {
    return !this.wide() && !this.open();
  }

  toggle(): void {
    this.open.update((v) => !v);
  }

  close(): void {
    this.open.set(false);
  }
}
