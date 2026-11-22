import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { Rental } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';
import { isOverdue } from '../../../../core/utils/charge-display';
import { Storage } from '../../../../core/models/storage';
import { StorageService } from '../../../../core/services/storage';
import { RequestService } from '../../../../core/services/request';
import { TenantRequest } from '../../../../core/models/request';
import { canBook } from '../../../../core/utils/resident-display';
import { rentalDebt } from '../../../../core/utils/rental-money';
import { LoadError } from '../../../../shared/components/load-error/load-error';

@Component({
  selector: 'app-client-dashboard',
  standalone: true,
  imports: [RouterLink, DatePipe, LoadError],
  templateUrl: './dashboard.html',
})
export class ClientDashboard {
  private auth = inject(AuthService);
  private rentalService = inject(RentalService);
  private storageService = inject(StorageService);
  private requestService = inject(RequestService);

  /** Tariff decreased: tenant still has to choose extend/refund. */
  pendingChoices = signal<TenantRequest[]>([]);

  transferInvitations = signal<Storage[]>([]);

  loading = signal(true);
  error = signal(false);
  rentals = signal<Rental[]>([]);

  // Индекс выбранной кладовой для карусели
  currentIndex = signal(0);

  // Переменные для отслеживания свайпа
  private touchStartX = 0;
  private touchEndX = 0;

  userName = computed(() => this.auth.currentUser()?.full_name || 'Житель');

  /** Pending / rejected / blocked residents can't book — no «Найти кладовку» for them. */
  bookable = computed(() => canBook(this.auth.currentUser()));
  notBookableText = computed(() => {
    switch (this.auth.currentUser()?.status) {
      case 'pending':
        return 'ОСИ проверяет, что вы собственник. Как только данные подтвердят, здесь можно будет арендовать кладовку.';
      case 'rejected':
        return 'ОСИ не подтвердил ваши данные. Исправьте их по подсказке вверху экрана и отправьте снова.';
      case 'blocked':
        return 'Аккаунт заблокирован ОСИ — новые аренды недоступны. Обратитесь в ОСИ вашего ЖК.';
      default:
        return 'Аренда пока недоступна. Обратитесь в ОСИ вашего ЖК.';
    }
  });

  /** ≈ debt of an overdue rental, same formula as the ОСИ ledger. */
  debt = (rental: Rental) => rentalDebt(rental);

  greeting = computed(() => {
    const hour = new Date().getHours();
    if (hour >= 5 && hour < 12) return 'Доброе утро';
    if (hour >= 12 && hour < 18) return 'Добрый день';
    if (hour >= 18 && hour < 24) return 'Добрый вечер';
    return 'Доброй ночи';
  });

  activeRentals = computed(() => this.rentals().filter((r) => r.status === 'active'));

  // Текущая активная кладовая с учетом карусели
  currentActiveRental = computed(() => {
    const list = this.activeRentals();
    if (list.length === 0) return null;
    return list[this.currentIndex()] || list[0];
  });

  // Методы обработки свайпов
  onTouchStart(event: TouchEvent): void {
    this.touchStartX = event.changedTouches[0].screenX;
  }

  onTouchEnd(event: TouchEvent): void {
    this.touchEndX = event.changedTouches[0].screenX;
    this.handleGesture();
  }

  private handleGesture(): void {
    const threshold = 50; // Минимальная длина свайпа в пикселях
    const list = this.activeRentals();
    if (!list || list.length <= 1) return;

    if (this.touchEndX < this.touchStartX - threshold) {
      // Свайп влево -> следующая кладовка
      this.nextStorage();
    }
    if (this.touchEndX > this.touchStartX + threshold) {
      // Свайп вправо -> предыдущая кладовка
      this.prevStorage();
    }
  }

  nextStorage() {
    const list = this.activeRentals();
    if (list.length <= 1) return;
    this.currentIndex.update((i) => (i + 1) % list.length);
  }

  prevStorage() {
    const list = this.activeRentals();
    if (list.length <= 1) return;
    this.currentIndex.update((i) => (i - 1 + list.length) % list.length);
  }

  lockedRentals = computed(() => this.rentals().filter((r) => r.status === 'locked'));

  /** Debt = a paid rental kept after its paid period (same rule as the server ledger). */
  hasDebt = computed(() => this.activeRentals().some((r) => isOverdue(r)));

  constructor() {
    this.load();
    this.requestService.listMine().subscribe({
      next: (list) => this.pendingChoices.set(list.filter((r) => r.status === 'awaiting_choice')),
      error: () => this.pendingChoices.set([]),
    });
  }

  statusLabel(status: string): string {
    switch (status) {
      case 'active':
        return 'Активна';
      case 'locked':
        return 'Ожидает оплаты';
      case 'expired':
        return 'Истекла';
      case 'cancelled':
        return 'Отменена';
      default:
        return status;
    }
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }

  reload(): void {
    this.load();
  }

  private load(): void {
    const user = this.auth.currentUser();
    const jkId = user?.jk_id;
    const userId = user?.id;

    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);

    this.rentalService.listMine().subscribe({
      next: (mine) => {
        this.rentals.set(mine);
        this.loading.set(false);
      },
      error: () => {
        this.rentals.set([]);
        this.loading.set(false);
        this.error.set(true);
      },
    });

    this.storageService.listByJK(jkId).subscribe({
      next: (storages) => {
        const userIin = this.auth.currentUser()?.iin;
        const now = Date.now();

        this.transferInvitations.set(
          storages.filter(
            (storage) =>
              !!userIin &&
              storage.transfer_to_iin === userIin &&
              !!storage.transfer_until &&
              new Date(storage.transfer_until).getTime() > now,
          ),
        );
      },
      error: () => this.transferInvitations.set([]),
    });
  }

  isOverdue(endsAt: string): boolean {
    return new Date(endsAt).getTime() < Date.now();
  }
}
