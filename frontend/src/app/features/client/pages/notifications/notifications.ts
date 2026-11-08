import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

import { NotificationList } from '../../../../shared/components/notification-list/notification-list';

@Component({
  selector: 'app-client-notifications',
  standalone: true,
  imports: [RouterLink, NotificationList],
  templateUrl: './notifications.html',
})
export class ClientNotifications {}
