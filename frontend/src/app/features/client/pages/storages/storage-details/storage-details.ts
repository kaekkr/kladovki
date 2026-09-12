import { DatePipe } from '@angular/common';
import { Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../../../core/auth/auth.service';
import { StorageService } from '../../../../../core/services/storage';
import { SettingsService } from '../../../../../core/services/settings';
import { Storage } from '../../../../../core/models/storage';
import { RentalService } from '../../../../../core/services/rental';

type RentalPeriod = 1 | 3 | 6 | 12;

@Component({
  selector: 'app-client-storage-details',
  standalone: true,
  imports: [RouterLink, DatePipe],
  templateUrl: './storage-details.html',
})
export class ClientStorageDetails {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private auth = inject(AuthService);
  private storageService = inject(StorageService);
  private settingsService = inject(SettingsService);
  private rentalService = inject(RentalService);

  renting = signal(false);
  rentalError = signal<string | null>(null);

  loading = signal(true);
  error = signal(false);

  storage = signal<Storage | null>(null);
  tariff = signal(0);

  rentalPeriods: RentalPeriod[] = [1, 3, 6, 12];

  selectedPeriod = signal<RentalPeriod>(1);

  pricePerMonth = computed(() => {
    const currentStorage = this.storage();

    if (!currentStorage || !this.tariff()) {
      return 0;
    }

    return currentStorage.area * this.tariff();
  });

  totalPrice = computed(() => {
    return this.pricePerMonth() * this.selectedPeriod();
  });

  constructor() {
    this.load();
  }

  selectPeriod(period: RentalPeriod): void {
    this.selectedPeriod.set(period);
  }

  formatMoney(value: number): string {
    return (
      new Intl.NumberFormat('ru-RU', {
        maximumFractionDigits: 0,
      }).format(value) + ' ₸'
    );
  }

  floorLabel(floor: number): string {
    if (floor === -1) {
      return 'Подземный этаж';
    }

    if (floor === 0) {
      return '1 этаж';
    }

    return `${floor} этаж`;
  }

  statusLabel(status: Storage['status']): string {
    switch (status) {
      case 'free':
        return 'Свободна';
      case 'locked':
        return 'Забронирована';
      case 'occupied':
        return 'Занята';
      default:
        return status;
    }
  }

  private load(): void {
    const storageId = this.route.snapshot.paramMap.get('id');
    const user = this.auth.currentUser();

    if (!storageId || !user?.jk_id) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.storageService.getById(storageId).subscribe({
      next: (storage) => {
        this.storage.set(storage);

        this.settingsService.getTariff(user.jk_id!).subscribe({
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
        this.storage.set(null);
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }

  rent(): void {
    const currentStorage = this.storage();

    if (!currentStorage || currentStorage.status !== 'free') {
      return;
    }

    this.renting.set(true);
    this.rentalError.set(null);

    this.rentalService.lockStorage(currentStorage.id, this.selectedPeriod()).subscribe({
      next: (rental) => {
        this.renting.set(false);

        // Пока просто переходим на страницу аренды.
        // Следующим шагом там сделаем оплату.
        // Предполагается, что rental-details route уже существует.
        this.router.navigate(['/app/rentals', rental.id]);
      },
      error: (error) => {
        this.renting.set(false);

        if (error.status === 409) {
          this.rentalError.set('Кладовка уже занята или забронирована.');

          // Обновляем данные кладовки,
          // чтобы UI получил актуальный status.
          this.load();
          return;
        }

        this.rentalError.set('Не удалось забронировать кладовку. Попробуйте ещё раз.');
      },
    });
  }
}
