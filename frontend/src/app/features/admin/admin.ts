import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AdminHeader } from './layout/header/header';
import { AdminSidebar } from './layout/sidebar/sidebar';
import { AdminFooter } from './layout/footer/footer';

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [RouterOutlet, AdminHeader, AdminSidebar, AdminFooter],
  templateUrl: './admin.html',
})
export class Admin { }
