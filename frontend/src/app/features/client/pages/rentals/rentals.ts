import { Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { Rental, RentalStatus } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';
import {
  rentalDisplayEndsAt,
  rentalStatusLabel,
  rentalTotalCost,
} from '../../../../core/utils/rental-display';
import { canBook } from '../../../../core/utils/resident-display';
import { formatRuDate, formatTenge } from '../../../../core/utils/date-format';
import { overdueDays, rentalPeriodLabel } from '../../../../core/utils/rental-money';
import { LoadError } from '../../../../shared/components/load-error/load-error';

@Component({
  selector: 'app-client-rentals',
  standalone: true,
  imports: [RouterLink, LoadError],
  templateUrl: './rentals.html',
})
export class ClientRentals {
  private auth = inject(AuthService);
  private rentalService = inject(RentalService);

  loading = signal(true);
  error = signal(false);
  rentals = signal<Rental[]>([]);

  activeRentals = computed(() => this.rentals().filter((rental) => rental.status === 'active'));

  historyRentals = computed(() =>
    this.rentals().filter((rental) => ['expired', 'cancelled'].includes(rental.status)),
  );

  constructor() {
    this.load();
  }

  statusLabel = rentalStatusLabel;
  periodLabel = rentalPeriodLabel;
  bookable = computed(() => canBook(this.auth.currentUser()));
  isOverdue = (rental: Rental) => overdueDays(rental) > 0;
  displayEndsAt = rentalDisplayEndsAt;
  totalCost = rentalTotalCost;

  statusClass(status: RentalStatus): string {
    switch (status) {
      case 'active':
        return 'bg-brand-accent/10 text-brand-accent';

      case 'locked':
        return 'bg-status-locked/10 text-status-locked';

      case 'expired':
      case 'cancelled':
        return 'bg-brand-elevated text-brand-muted';

      default:
        return 'bg-brand-elevated text-brand-muted';
    }
  }

  formatMoney = formatTenge;
  formatDate = formatRuDate;

  load(): void {
    const user = this.auth.currentUser();
    const jkId = user?.jk_id;
    const userId = user?.id;

    if (!jkId || !userId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.rentalService.listMine().subscribe({
      next: (myRentals) => {
        this.rentals.set(myRentals);
        this.loading.set(false);
      },

      error: () => {
        this.rentals.set([]);
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }
}
