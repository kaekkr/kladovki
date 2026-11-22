import { Component, HostListener, inject } from '@angular/core';
import { AdminNav } from '../admin-nav';
import { UserProfile } from '../../../../shared/components/user_profile/user_profile';
import { AdminNavbar } from '../navbar/navbar';

@Component({
  selector: 'app-admin-sidebar',
  standalone: true,
  imports: [UserProfile, AdminNavbar],
  templateUrl: './sidebar.html',
})
export class AdminSidebar {
  nav = inject(AdminNav);

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.nav.close();
  }
}
