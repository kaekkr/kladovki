import { Component } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { AdminNavbar } from '../navbar/navbar';

@Component({
  selector: 'app-admin-sidebar',
  standalone: true,
  imports: [RouterLink, RouterLinkActive, AdminNavbar],
  templateUrl: './sidebar.html',
})
export class AdminSidebar { }
