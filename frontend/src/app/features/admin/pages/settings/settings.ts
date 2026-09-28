import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../../../core/auth/auth.service';
import { SettingsService } from '../../../../core/services/settings';
import { AdminSettings } from '../../../../core/models/settings';

@Component({
  selector: 'app-admin-settings',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './settings.html',
})
export class AdminSettingsPage {
  Math = Math;

  private authService = inject(AuthService);
  private settingsService = inject(SettingsService);

  loading = signal(true);
  error = signal(false);
  saving = signal(false);
  saveMessage = signal('');
  saveError = signal('');

  // Form state
  jkId = signal('');
  jkName = signal('');
  jkAddress = signal('');
  tariffAmount = signal(0);
  initialTariffAmount = signal(0); // Для отслеживания уменьшения
  lockMinutes = signal(2);

  // Read-only account
  userName = signal('');
  userPhone = signal('');

  // Состояние для копирования ссылки
  copied = signal(false);

  // Модальное окно перерасчета при уменьшении тарифа
  showRecalculateModal = signal(false);
  affectedRentalsCount = signal(0);
  totalDifferenceAmount = signal(0);
  recalculationStrategy = signal<'extend' | 'refund'>('extend');

  registrationLink = computed(() => {
    const id = this.jkId();
    if (!id) return '';
    const origin = window.location.origin;
    return `${origin}/auth/register?jk=${id}`;
  });

  getQrCodeUrl(): string {
    const link = this.registrationLink();
    return `https://api.qrserver.com/v1/create-qr-code/?size=100x100&data=${encodeURIComponent(link)}`;
  }

  copyLink() {
    const link = this.registrationLink();
    if (!link) return;
    navigator.clipboard.writeText(link);
    this.copied.set(true);
    setTimeout(() => this.copied.set(false), 2000);
  }

  constructor() {
    this.load();
  }

  private load(): void {
    const user = this.authService.currentUser();
    const jkId = user?.jk_id;
    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }

    this.jkId.set(jkId);
    this.userName.set(user?.full_name ?? 'Админ');
    this.userPhone.set(user?.phone ?? '');

    this.loading.set(true);
    this.error.set(false);

    this.settingsService.load(jkId).subscribe({
      next: (s: AdminSettings) => {
        this.jkName.set(s.jk.name);
        this.jkAddress.set(s.jk.address ?? '');
        this.tariffAmount.set(s.tariff.amount);
        this.initialTariffAmount.set(s.tariff.amount); // Сохраняем оригинал
        this.lockMinutes.set(s.rental.lock_duration_minutes);
        this.loading.set(false);
      },
      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  save(): void {
    const newTariff = Number(this.tariffAmount()) || 0;
    const oldTariff = Number(this.initialTariffAmount()) || 0;

    // Если тариф уменьшился — запрашиваем превью с бэкенда
    if (newTariff < oldTariff) {
      const jkId = this.authService.currentUser()?.jk_id;
      if (!jkId) return;

      this.saving.set(true);
      this.settingsService.previewTariffReduction(jkId, newTariff).subscribe({
        next: (res) => {
          this.saving.set(false);
          // Подставляем реальные данные из бэкенда в сигналы модального окна
          this.affectedRentalsCount.set(res.affected_count);
          this.totalDifferenceAmount.set(res.total_difference);
          this.showRecalculateModal.set(true);
        },
        error: () => {
          this.saving.set(false);
          this.saveError.set('Не удалось рассчитать перерасчет');
        },
      });
      return;
    }

    // Если тариф вырос или не изменился — сохраняем напрямую
    this.executeSave();
  }

  confirmRecalculation(): void {
    this.showRecalculateModal.set(false);
    this.executeSave(this.recalculationStrategy());
  }

  executeSave(strategy?: 'extend' | 'refund'): void {
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) return;

    this.saving.set(true);
    this.saveMessage.set('');
    this.saveError.set('');

    const amount = Number(this.tariffAmount()) || 0;
    const lock = Math.max(1, Number(this.lockMinutes()) || 2);

    this.settingsService.saveTariff(jkId, amount, strategy).subscribe({
      next: () => {
        this.settingsService.saveRentalSettings(jkId, { lock_duration_minutes: lock }).subscribe({
          next: () => {
            this.saving.set(false);
            this.initialTariffAmount.set(amount); // Обновляем базовый тариф
            this.saveMessage.set('Настройки и перерасчет успешно сохранены');
            setTimeout(() => this.saveMessage.set(''), 3000);
          },
          error: () => {
            this.saving.set(false);
            this.saveError.set('Не удалось сохранить настройки аренды');
          },
        });
      },
      error: () => {
        this.saving.set(false);
        this.saveError.set('Не удалось сохранить тариф');
      },
    });
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }
}
