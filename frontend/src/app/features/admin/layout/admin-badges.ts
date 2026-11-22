import { Injectable, inject, signal } from '@angular/core';

import { RequestService } from '../../../core/services/request';
import { ResidentService } from '../../../core/services/resident';

/** Sidebar counters; pages call refresh() after resolving a заявка or reviewing a resident. */
@Injectable({ providedIn: 'root' })
export class AdminBadges {
  private requests = inject(RequestService);
  private residents = inject(ResidentService);

  readonly openRequests = signal(0);
  readonly pendingResidents = signal(0);

  refresh(): void {
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
