import { Component, OnInit, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../core/auth/auth.service';

/** Sidebar profile block of the ОСИ panel: name, role and a menu with logout. */
@Component({
  selector: 'app-user-profile',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './user_profile.html',
})
export class UserProfile implements OnInit {
  private authService = inject(AuthService);
  private router = inject(Router);

  currentUser = this.authService.currentUser;
  menuOpen = signal(false);

  ngOnInit(): void {
    if (!this.currentUser()) {
      this.authService.getMe().subscribe();
    }
  }

  get roleLabel(): string {
    return this.currentUser()?.role === 'admin' ? 'Администратор ОСИ' : 'Житель';
  }

  get initials(): string {
    const name = this.currentUser()?.full_name?.trim();
    if (!name) return 'U';

    return name
      .split(/\s+/)
      .map((part) => part[0].toUpperCase())
      .slice(0, 2)
      .join('');
  }

  logout(): void {
    this.menuOpen.set(false);
    this.authService.logout().subscribe(() => this.router.navigate(['/login']));
  }
}
