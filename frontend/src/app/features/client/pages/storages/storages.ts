import { Component, computed, inject, signal } from '@angular/core';
import { floorLabel } from '../../../../core/utils/floor';
import { storageStatusLabel } from '../../../../core/utils/storage-status';
import { RouterLink } from '@angular/router';
import { Storage } from '../../../../core/models/storage';
import { StorageService } from '../../../../core/services/storage';
import { SettingsService } from '../../../../core/services/settings';
import { AuthService } from '../../../../core/auth/auth.service';
import { formatArea } from '../../../../core/utils/area';
import { formatTenge } from '../../../../core/utils/date-format';
import { renewalPrice } from '../../../../core/utils/rental-display';
import { LoadError } from '../../../../shared/components/load-error/load-error';

type StorageFilter = 'all' | 'free';

@Component({
  selector: 'app-client-storages',
  standalone: true,
  imports: [RouterLink, LoadError],
  templateUrl: './storages.html',
})
export class ClientStorages {
  private auth = inject(AuthService);
  private storageService = inject(StorageService);
  private settingsService = inject(SettingsService);

  loading = signal(true);
  error = signal(false);

  storages = signal<Storage[]>([]);
  tariff = signal(0);
  filter = signal<StorageFilter>('all');

  filteredStorages = computed(() => {
    const currentFilter = this.filter();

    if (currentFilter === 'free') {
      return this.storages().filter((storage) => storage.status === 'free');
    }

    return this.storages();
  });

  freeCount = computed(() => this.storages().filter((storage) => storage.status === 'free').length);

  occupiedCount = computed(
    () => this.storages().filter((storage) => storage.status === 'occupied').length,
  );

  lockedCount = computed(
    () => this.storages().filter((storage) => storage.status === 'locked').length,
  );

  floors = computed(() => {
    const grouped = new Map<number, Storage[]>();

    for (const storage of this.filteredStorages()) {
      const floor = storage.floor;

      if (!grouped.has(floor)) {
        grouped.set(floor, []);
      }

      grouped.get(floor)!.push(storage);
    }

    return Array.from(grouped.entries())
      .sort(([floorA], [floorB]) => floorB - floorA)
      .map(([floor, storages]) => ({
        floor,
        storages: [...storages].sort((a, b) =>
          a.number.localeCompare(b.number, undefined, {
            numeric: true,
            sensitivity: 'base',
          }),
        ),
      }));
  });

  constructor() {
    this.load();
  }

  setFilter(filter: StorageFilter): void {
    this.filter.set(filter);
  }

  /** Same as the server: round(area × tariff). */
  pricePerMonth(storage: Storage): number {
    return renewalPrice(storage.area, this.tariff(), 1);
  }

  formatMoney = formatTenge;
  area = formatArea;

  statusLabel = storageStatusLabel;

  floorLabel = floorLabel;

  load(): void {
    const user = this.auth.currentUser();
    const jkId = user?.jk_id;

    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.storageService.listByJK(jkId).subscribe({
      next: (storages) => {
        this.storages.set(storages);

        this.settingsService.getTariff(jkId).subscribe({
          next: (tariff) => {
            this.tariff.set(tariff.amount);
            this.loading.set(false);
          },
          error: () => {
            this.tariff.set(0);
            this.loading.set(false);
          },
        });
      },
      error: () => {
        this.storages.set([]);
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }
}
