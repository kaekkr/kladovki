import { Component, computed, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../../../core/auth/auth.service';

@Component({
  selector: 'app-client-profile',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './profile.html',
})
export class ClientProfile {
  private auth = inject(AuthService);
  private router = inject(Router);

  user = computed(() => this.auth.currentUser());

  initials = computed(() => {
    const name = this.user()?.full_name?.trim();

    if (!name) {
      return 'Ж';
    }

    return name
      .split(/\s+/)
      .slice(0, 2)
      .map((part) => part.charAt(0))
      .join('')
      .toUpperCase();
  });

  logout(): void {
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}
