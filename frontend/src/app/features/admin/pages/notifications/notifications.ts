import { Component } from '@angular/core';

import { NotificationList } from '../../../../shared/components/notification-list/notification-list';

@Component({
  selector: 'app-admin-notifications',
  standalone: true,
  imports: [NotificationList],
  template: `
    <div class="max-w-2xl space-y-6">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-brand-text">Уведомления</h1>
        <p class="mt-1 text-sm text-brand-muted">Просрочки, новые заявки и возвраты</p>
      </div>
      <app-notification-list />
    </div>
  `,
})
export class AdminNotifications {}
