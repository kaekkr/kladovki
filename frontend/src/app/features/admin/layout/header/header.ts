import { Component, OnInit, inject, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { JkService } from '../../../../core/services/jk';

@Component({
  selector: 'app-admin-header',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './header.html',
})
export class AdminHeader implements OnInit {
  private authService = inject(AuthService);
  private jkService = inject(JkService);

  currentJk = this.jkService.currentJk;

  constructor() {
    effect(() => {
      const user = this.authService.currentUser();
      if (user?.jk_id && !this.currentJk()) {
        this.jkService.getJKById(user.jk_id).subscribe();
      }
    });
  }

  ngOnInit(): void {
    const user = this.authService.currentUser();
    if (user?.jk_id && !this.currentJk()) {
      this.jkService.getJKById(user.jk_id).subscribe();
    }
  }
}
