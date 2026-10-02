import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';
import { Rental } from '../../../../core/models/rental';
import { RentalService } from '../../../../core/services/rental';

@Component({
  selector: 'app-client-dashboard',
  standalone: true,
  imports: [RouterLink, DatePipe],
  templateUrl: './dashboard.html',
})
export class ClientDashboard {
  private auth = inject(AuthService);
  private rentalService = inject(RentalService);

  loading = signal(true);
  error = signal(false);
  rentals = signal<Rental[]>([]);

  // Индекс выбранной кладовой для карусели
  currentIndex = signal(0);

  // Переменные для отслеживания свайпа
  private touchStartX = 0;
  private touchEndX = 0;

  userName = computed(() => this.auth.currentUser()?.full_name || 'Житель');

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

  hasDebt = computed(() =>
    this.activeRentals().some((r) => r.total_paid < r.price_per_month * r.months),
  );

  constructor() {
    this.load();
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

    this.rentalService.listByJK(jkId).subscribe({
      next: (all) => {
        const mine = userId ? all.filter((r) => r.user_id === userId) : [];
        this.rentals.set(mine);
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
