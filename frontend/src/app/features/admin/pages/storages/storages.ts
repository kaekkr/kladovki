import { Component, computed, inject, signal } from '@angular/core';

import { AuthService } from '../../../../core/auth/auth.service';
import { Storage, StorageStatus } from '../../../../core/models/storage';
import { StorageService } from '../../../../core/services/storage';
import { AdminStorageDetailsModal } from './components/storage_details_modal/storage_details_modal';
import { AdminStorageCreateModal } from './components/storage_create_modal/storage_create_modal';

@Component({
  selector: 'app-admin-storages',
  standalone: true,
  imports: [AdminStorageDetailsModal, AdminStorageCreateModal],
  templateUrl: './storages.html',
})
export class AdminStorages {
  private authService = inject(AuthService);
  private storageService = inject(StorageService);

  storages = signal<Storage[]>([]);
  loading = signal(true);
  error = signal(false);

  deleteTarget = signal<Storage | null>(null);
  deleting = signal(false);
  deleteError = signal<string | null>(null);

  createModalOpen = signal(false);
  creating = signal(false);
  createError = signal<string | null>(null);

  search = signal('');
  statusFilter = signal<StorageStatus | 'all'>('all');

  selectedStorage = signal<Storage | null>(null);

  filteredStorages = computed(() => {
    const search = this.search().trim().toLowerCase();
    const status = this.statusFilter();

    return this.storages().filter((storage) => {
      const matchesSearch = !search || storage.number.toLowerCase().includes(search);

      const matchesStatus = status === 'all' || storage.status === status;

      return matchesSearch && matchesStatus;
    });
  });

  constructor() {
    this.loadStorages();
  }

  private loadStorages(): void {
    const jkId = this.authService.currentUser()?.jk_id;

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
        this.loading.set(false);
      },

      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  setSearch(value: string): void {
    this.search.set(value);
  }

  setStatusFilter(status: StorageStatus | 'all'): void {
    this.statusFilter.set(status);
  }

  statusLabel(status: StorageStatus): string {
    switch (status) {
      case 'free':
        return 'Свободна';

      case 'occupied':
        return 'Занята';

      case 'locked':
        return 'Заблокирована';

      default:
        return status;
    }
  }

  openStorage(storage: Storage) {
    this.selectedStorage.set(storage);
  }

  closeStorage() {
    this.selectedStorage.set(null);
  }

  updateStorage(updated: Storage): void {
    this.storageService
      .update(updated.id, {
        number: updated.number,
        area: updated.area,
        floor: updated.floor,
        entrance: updated.entrance,
      })
      .subscribe({
        next: (storage) => {
          this.storages.update((items) =>
            items.map((item) => (item.id === storage.id ? storage : item)),
          );

          this.selectedStorage.set(storage);
        },

        error: () => {
          // здесь позже сделаем нормальный toast
        },
      });
  }

  openDelete(storage: Storage): void {
    this.deleteError.set(null);
    this.deleteTarget.set(storage);
  }

  closeDelete(): void {
    if (this.deleting()) {
      return;
    }

    this.deleteTarget.set(null);
    this.deleteError.set(null);
  }

  deleteStorage(): void {
    const storage = this.deleteTarget();

    if (!storage) {
      return;
    }

    this.deleting.set(true);
    this.deleteError.set(null);

    this.storageService.delete(storage.id).subscribe({
      next: () => {
        this.storages.update((items) => items.filter((item) => item.id !== storage.id));

        this.selectedStorage.set(null);
        this.deleteTarget.set(null);
        this.deleting.set(false);
      },

      error: (err) => {
        this.deleting.set(false);

        if (err.status === 409) {
          this.deleteError.set('Нельзя удалить кладовую, которая связана с арендой.');
          return;
        }

        this.deleteError.set('Не удалось удалить кладовую. Попробуйте ещё раз.');
      },
    });
  }

  openCreate(): void {
    this.createError.set(null);
    this.createModalOpen.set(true);
  }

  closeCreate(): void {
    if (this.creating()) {
      return;
    }

    this.createModalOpen.set(false);
    this.createError.set(null);
  }

  createStorage(storage: Storage): void {
    const jkId = this.authService.currentUser()?.jk_id;

    if (!jkId) {
      this.createError.set('Не удалось определить ЖК.');
      return;
    }

    this.creating.set(true);
    this.createError.set(null);

    this.storageService
      .create({
        jk_id: jkId,
        number: storage.number,
        area: storage.area,
        floor: storage.floor,
        entrance: storage.entrance,
      })
      .subscribe({
        next: (created) => {
          this.storages.update((items) => [...items, created]);

          this.creating.set(false);
          this.createModalOpen.set(false);
        },

        error: (err) => {
          this.creating.set(false);

          if (err.status === 409) {
            this.createError.set('Кладовая с таким номером уже существует.');
            return;
          }

          this.createError.set('Не удалось создать кладовую. Попробуйте ещё раз.');
        },
      });
  }
}
