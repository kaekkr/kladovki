import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { RegisterPayload } from '../../../core/auth/auth.models';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [FormsModule, RouterLink],
  templateUrl: './register.html',
})
export class Register {
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private authService = inject(AuthService);

  // Form fields
  fullName = signal('');
  phone = signal('');
  password = signal('');
  acceptedRules = signal(false);

  // State
  loading = signal(false);
  errorMessage = signal('');

  // Состояние модального окна с регламентом
  showTerms = signal(false);

  // ЖК ID из query параметров (например, /auth/register?jk=UUID)
  jkId = signal(this.route.snapshot.queryParamMap.get('jk') ?? '');

  register(): void {
    if (!this.fullName() || !this.phone() || !this.password()) {
      this.errorMessage.set('Заполните все обязательные поля');
      return;
    }

    if (!this.acceptedRules()) {
      this.errorMessage.set('Необходимо принять регламент использования сервиса');
      return;
    }

    this.loading.set(true);
    this.errorMessage.set('');

    const payload: RegisterPayload = {
      full_name: this.fullName(),
      phone: this.phone(),
      password: this.password(),
      jk_id: this.jkId() || undefined,
    };

    this.authService.register(payload).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/app']);
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMessage.set(err.error?.message || 'Ошибка регистрации. Попробуйте позже.');
      },
    });
  }
}
