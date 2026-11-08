import { Component, OnInit, inject, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { JkService } from '../../../../core/services/jk';
import { NotificationService } from '../../../../core/services/notification';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-admin-header',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './header.html',
})
export class AdminHeader implements OnInit {
  private authService = inject(AuthService);
  private jkService = inject(JkService);

  currentJk = this.jkService.currentJk;
  private notifications = inject(NotificationService);
  unread = this.notifications.unread;

  constructor() {
    effect(() => {
      const user = this.authService.currentUser();
      if (user?.jk_id && !this.currentJk()) {
        this.jkService.getJKById(user.jk_id).subscribe();
      }
    });
  }

  ngOnInit(): void {
    this.notifications.refreshUnread();
    const user = this.authService.currentUser();
    if (user?.jk_id && !this.currentJk()) {
      this.jkService.getJKById(user.jk_id).subscribe();
    }
  }
}
