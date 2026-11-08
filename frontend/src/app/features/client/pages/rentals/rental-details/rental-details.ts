import { Component, computed, inject, OnDestroy, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { Rental, RentalStatus } from '../../../../../core/models/rental';
import { RentalService } from '../../../../../core/services/rental';
import { StorageService } from '../../../../../core/services/storage';
import { RequestService } from '../../../../../core/services/request';
import { SettingsService } from '../../../../../core/services/settings';
import { TenantRequest } from '../../../../../core/models/request';
import { requestStatusLabel, requestTitle } from '../../../../../core/utils/request-display';
import {
  canRenew,
  isUnpaidExpired,
  renewalEndsAt,
  renewalPrice,
  rentalDisplayEndsAt,
  rentalStatusLabel,
  rentalTotalCost,
} from '../../../../../core/utils/rental-display';

@Component({
  selector: 'app-client-rental-details',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './rental-details.html',
})
export class ClientRentalDetails implements OnDestroy {
  private route = inject(ActivatedRoute);
  private rentalService = inject(RentalService);
  private storageService = inject(StorageService);
  private requestService = inject(RequestService);
  private settingsService = inject(SettingsService);

  // Renewal (продление): pay for more months on an active rental.
  canRenew = canRenew;
  renewOptions = [1, 3, 6, 12];
  renewMonths = signal(1);
  renewing = signal(false);
  renewError = signal<string | null>(null);
  renewMessage = signal<string | null>(null);
  private storageArea = signal<number | null>(null);
  private tariff = signal<number | null>(null);
  private renewPoll: ReturnType<typeof setInterval> | null = null;

  renewEnds = computed(() => {
    const r = this.rental();
    return r ? renewalEndsAt(r, this.renewMonths()) : '';
  });
  renewPriceValue = computed(() => {
    const area = this.storageArea();
    const tariff = this.tariff();
    return area === null || tariff === null ? null : renewalPrice(area, tariff, this.renewMonths());
  });

  requests = signal<TenantRequest[]>([]);
  requestTitle = requestTitle;
  requestStatusLabel = requestStatusLabel;
  choosing = signal(false);
  requestError = signal<string | null>(null);
  terminateOpen = signal(false);
  terminateComment = signal('');

  pendingChoice = computed(() => this.requests().find((r) => r.status === 'awaiting_choice') ?? null);
  openTermination = computed(
    () =>
      this.requests().find((r) => r.type === 'early_termination' && r.status === 'open') ?? null,
  );

  loading = signal(true);
  error = signal(false);
  rental = signal<Rental | null>(null);
  transferIIN = signal('');
  transferLoading = signal(false);
  transferMessage = signal<string | null>(null);
  transferError = signal<string | null>(null);

  paying = signal(false); // Теперь это состояние "Ожидание оплаты в Kaspi"
  paymentError = signal<string | null>(null);

  remainingSeconds = signal(0);

  private timer: ReturnType<typeof setInterval> | null = null;
  private pollTimer: ReturnType<typeof setInterval> | null = null;

  constructor() {
    this.load();
  }

  isUnpaidExpired = isUnpaidExpired;
  totalCost = rentalTotalCost;
  displayEndsAt = rentalDisplayEndsAt;
  displayStatusLabel = rentalStatusLabel;

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

  formatMoney(value: number): string {
    return (
      new Intl.NumberFormat('ru-RU', {
        maximumFractionDigits: 0,
      }).format(value) + ' ₸'
    );
  }

  formatDate(value: string): string {
    return new Intl.DateTimeFormat('ru-RU', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    }).format(new Date(value));
  }

  formatTime(seconds: number): string {
    const minutes = Math.floor(seconds / 60);
    const remaining = seconds % 60;

    return `${String(minutes).padStart(2, '0')}:${String(remaining).padStart(2, '0')}`;
  }

  setTransfer() {
    const rental = this.rental();
    const iin = this.transferIIN().trim();

    if (!rental || !/^\d{12}$/.test(iin)) {
      this.transferError.set('ИИН должен содержать 12 цифр');
      return;
    }

    this.transferLoading.set(true);
    this.transferError.set(null);
    this.transferMessage.set(null);

    this.storageService.setTransfer(rental.storage_id, iin).subscribe({
      next: () => {
        this.transferMessage.set('Новый владелец указан успешно');
        this.transferIIN.set('');
        this.transferLoading.set(false);
      },
      error: (err) => {
        this.transferError.set(err.error?.error || 'Не удалось указать нового владельца');
        this.transferLoading.set(false);
      },
    });
  }

  canPay(): boolean {
    return this.rental()?.status === 'locked' && this.remainingSeconds() > 0 && !this.paying();
  }

  pay(): void {
    const currentRental = this.rental();

    if (!currentRental || !this.canPay()) {
      return;
    }

    this.paying.set(true);
    this.paymentError.set(null);

    this.rentalService
      .createPayment({ rental_id: currentRental.id })
      .subscribe({
        next: () => {
          // Счет успешно отправлен в Kaspi, запускаем опрос (поллинг) статуса
          this.startPolling(currentRental.id);
        },
        error: (error) => {
          this.paying.set(false);

          if (error.status === 409) {
            this.paymentError.set(
              'Срок бронирования истёк. Кладовка больше не удерживается за вами.',
            );
            this.load();
            return;
          }

          this.paymentError.set(
            error.error?.error || 'Не удалось отправить счёт в Kaspi. Попробуйте ещё раз.',
          );
        },
      });
  }

  // Запуск поллинга для ожидания оплаты
  private startPolling(rentalId: string): void {
    this.stopPolling();

    this.pollTimer = setInterval(() => {
      this.rentalService.getById(rentalId).subscribe({
        next: (rental) => {
          if (rental.status === 'active') {
            // Оплата прошла успешно!
            this.stopPolling();
            this.paying.set(false);
            this.rental.set(rental);
            // Можно обновить таймеры или перенаправить/показать успех
          } else if (rental.status === 'cancelled' || rental.status === 'expired') {
            // Если бронь отменилась или истекла
            this.stopPolling();
            this.paying.set(false);
            this.rental.set(rental);
            this.paymentError.set('Время оплаты истекло, бронь отменена.');
          }
        },
        error: () => {
          // Игнорируем сетевые ошибки во время поллинга, пробуем снова
        },
      });
    }, 3000); // Проверяем каждые 3 секунды
  }

  private stopPolling(): void {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
  }

  private startTimer(endsAt: string): void {
    this.stopTimer();

    const update = () => {
      const end = new Date(endsAt).getTime();
      const now = Date.now();

      const seconds = Math.max(0, Math.ceil((end - now) / 1000));

      this.remainingSeconds.set(seconds);

      if (seconds <= 0) {
        this.stopTimer();

        if (this.rental()?.status === 'locked') {
          this.load();
        }
      }
    };

    update();

    this.timer = setInterval(update, 1000);
  }

  private stopTimer(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }

  private load(): void {
    const rentalId = this.route.snapshot.paramMap.get('id');

    if (!rentalId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.loading.set(true);
    this.error.set(false);
    this.paymentError.set(null);

    this.rentalService.getById(rentalId).subscribe({
      next: (rental) => {
        this.rental.set(rental);
        this.loading.set(false);
        this.loadRequests(rental.id);
        if (canRenew(rental)) this.loadRenewalPrice(rental);

        if (rental.status === 'locked') {
          this.startTimer(rental.ends_at);
        } else {
          this.stopTimer();
          this.remainingSeconds.set(0);
        }
      },
      error: () => {
        this.rental.set(null);
        this.loading.set(false);
        this.error.set(true);
        this.stopTimer();
      },
    });
  }

  private loadRequests(rentalId: string): void {
    this.requestService.listMine().subscribe({
      next: (list) => this.requests.set(list.filter((r) => r.rental_id === rentalId)),
      error: () => this.requests.set([]),
    });
  }

  choose(choice: 'extend' | 'refund'): void {
    const req = this.pendingChoice();
    const rental = this.rental();
    if (!req || !rental) return;
    this.choosing.set(true);
    this.requestError.set(null);
    this.requestService.choose(req.id, choice).subscribe({
      next: () => {
        this.choosing.set(false);
        this.load();
      },
      error: (err) => {
        this.choosing.set(false);
        this.requestError.set(err?.error?.error || 'Не удалось сохранить выбор');
      },
    });
  }

  canTerminate(): boolean {
    const r = this.rental();
    return !!r && r.status === 'active' && !this.isOverdue(r.ends_at) && !this.openTermination();
  }

  terminate(): void {
    const rental = this.rental();
    if (!rental) return;
    this.choosing.set(true);
    this.requestError.set(null);
    this.requestService.terminateRental(rental.id, this.terminateComment().trim()).subscribe({
      next: () => {
        this.choosing.set(false);
        this.terminateOpen.set(false);
        this.loadRequests(rental.id);
      },
      error: (err) => {
        this.choosing.set(false);
        this.requestError.set(err?.error?.error || 'Не удалось отправить заявку');
      },
    });
  }

  private loadRenewalPrice(rental: Rental): void {
    this.storageService.getById(rental.storage_id).subscribe({
      next: (st) => this.storageArea.set(st.area),
      error: () => this.storageArea.set(null),
    });
    this.settingsService.getTariff(rental.jk_id).subscribe({
      next: (t) => this.tariff.set(t.amount > 0 ? t.amount : null),
      error: () => this.tariff.set(null),
    });
  }

  renew(): void {
    const rental = this.rental();
    if (!rental || this.renewing()) return;
    const previousEnd = rental.ends_at;
    this.renewing.set(true);
    this.renewError.set(null);
    this.renewMessage.set(null);

    this.rentalService.renew(rental.id, this.renewMonths()).subscribe({
      next: () => this.pollRenewal(rental.id, previousEnd),
      error: (err) => {
        this.renewing.set(false);
        this.renewError.set(err?.error?.error || 'Не удалось отправить счёт в Kaspi. Попробуйте ещё раз.');
      },
    });
  }

  /** Waits (up to 15 min) until the webhook moves ends_at forward. */
  private pollRenewal(rentalId: string, previousEnd: string): void {
    this.stopRenewPoll();
    const deadline = Date.now() + 15 * 60_000;
    const check = () => {
      this.rentalService.getById(rentalId).subscribe({
        next: (r) => {
          if (r.ends_at !== previousEnd) {
            this.stopRenewPoll();
            this.renewing.set(false);
            this.rental.set(r);
            this.renewMessage.set(`Оплачено. Аренда продлена до ${this.formatDate(r.ends_at)}.`);
          } else if (Date.now() > deadline) {
            this.stopRenewPoll();
            this.renewing.set(false);
            this.renewError.set('Оплата пока не поступила. Если вы оплатили — обновите страницу позже.');
          }
        },
      });
    };
    check();
    this.renewPoll = setInterval(check, 3000);
  }

  private stopRenewPoll(): void {
    if (this.renewPoll) {
      clearInterval(this.renewPoll);
      this.renewPoll = null;
    }
  }

  cancelPaymentLoading(): void {
    this.stopPolling();
    this.paying.set(false);
  }

  isOverdue(endsAt: string): boolean {
    return new Date(endsAt).getTime() < Date.now();
  }

  ngOnDestroy(): void {
    this.stopTimer();
    this.stopPolling();
    this.stopRenewPoll();
  }
}
