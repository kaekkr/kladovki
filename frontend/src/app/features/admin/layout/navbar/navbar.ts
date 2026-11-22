import { Component, DestroyRef, inject } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { filter } from 'rxjs';

import { AdminBadges } from '../admin-badges';

@Component({
  selector: 'app-admin-navbar',
  standalone: true,
  imports: [RouterLink, RouterLinkActive],
  templateUrl: './navbar.html',
  host: {
    class: 'flex-grow flex flex-col min-h-0',
  },
})
export class AdminNavbar {
  private badges = inject(AdminBadges);
  openRequests = this.badges.openRequests;
  pendingResidents = this.badges.pendingResidents;

  constructor() {
    // Initial load + after each navigation. The navbar is created during the
    // first navigation, so its NavigationEnd covers the initial load too.
    inject(Router)
      .events.pipe(
        filter((e) => e instanceof NavigationEnd),
        takeUntilDestroyed(inject(DestroyRef)),
      )
      .subscribe(() => this.badges.refresh());
  }
}
