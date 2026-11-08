import { Component, inject } from '@angular/core';
import { NotificationService } from '../../../../core/services/notification';
import { AuthService } from '../../../../core/auth/auth.service';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-client-header',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './header.html',
})
export class ClientHeader {
  private auth = inject(AuthService);

  private notifications = inject(NotificationService);

  name = this.auth.currentUser()?.full_name || 'Житель';
  unread = this.notifications.unread;

  constructor() {
    this.notifications.refreshUnread();
  }
}
