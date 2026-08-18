import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AdminHeader } from './components/header/header';
import { AdminSidebar } from './components/sidebar/sidebar';
import { AdminFooter } from './components/footer/footer';

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [RouterOutlet, AdminHeader, AdminSidebar, AdminFooter],
  templateUrl: './admin.html',
})
export class Admin { }
