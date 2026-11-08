import { Component, DestroyRef, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { filter } from 'rxjs';

import { RequestService } from '../../../../core/services/request';
import { ResidentService } from '../../../../core/services/resident';

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
  private requests = inject(RequestService);
  private residents = inject(ResidentService);
  openRequests = signal(0);
  pendingResidents = signal(0);

  constructor() {
    this.refresh();
    // Refresh the badge after each navigation (e.g. after resolving a request).
    inject(Router)
      .events.pipe(
        filter((e) => e instanceof NavigationEnd),
        takeUntilDestroyed(inject(DestroyRef)),
      )
      .subscribe(() => this.refresh());
  }

  private refresh(): void {
    this.requests.openCount().subscribe({
      next: (r) => this.openRequests.set(r.open),
      error: () => this.openRequests.set(0),
    });
    this.residents.pendingCount().subscribe({
      next: (r) => this.pendingResidents.set(r.pending),
      error: () => this.pendingResidents.set(0),
    });
  }
}
