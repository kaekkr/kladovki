import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../../core/auth/auth.service';

@Component({
  selector: 'app-user-profile',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './user_profile.html',
})
export class UserProfile implements OnInit {
  private authService = inject(AuthService);

  currentUser = this.authService.currentUser;

  ngOnInit(): void {
    if (!this.currentUser()) {
      this.authService.getMe().subscribe();
    }
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
}
