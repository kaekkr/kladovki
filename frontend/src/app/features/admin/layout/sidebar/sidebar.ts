import { Component } from '@angular/core';
import { UserProfile } from '../../../../shared/components/user_profile/user_profile';
import { AdminNavbar } from '../navbar/navbar';

@Component({
  selector: 'app-admin-sidebar',
  standalone: true,
  imports: [UserProfile, AdminNavbar],
  templateUrl: './sidebar.html',
})
export class AdminSidebar { }
