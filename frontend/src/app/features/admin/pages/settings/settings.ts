import { Component, inject, signal } from '@angular/core';
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
  jkName = signal('');
  jkAddress = signal('');
  tariffAmount = signal(0);
  lockMinutes = signal(2);

  // Read-only account
  userName = signal('');
  userPhone = signal('');

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

    this.userName.set(user?.full_name ?? 'Админ');
    this.userPhone.set(user?.phone ?? '');

    this.loading.set(true);
    this.error.set(false);

    this.settingsService.load(jkId).subscribe({
      next: (s: AdminSettings) => {
        this.jkName.set(s.jk.name);
        this.jkAddress.set(s.jk.address ?? '');
        this.tariffAmount.set(s.tariff.amount);
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
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) return;

    this.saving.set(true);
    this.saveMessage.set('');
    this.saveError.set('');

    const amount = Number(this.tariffAmount()) || 0;
    const lock = Math.max(1, Number(this.lockMinutes()) || 2);

    // Save tariff + rental settings (parallel when both exist)
    this.settingsService.saveTariff(jkId, amount).subscribe({
      next: () => {
        this.settingsService.saveRentalSettings(jkId, { lock_duration_minutes: lock }).subscribe({
          next: () => {
            this.saving.set(false);
            this.saveMessage.set('Настройки сохранены');
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
