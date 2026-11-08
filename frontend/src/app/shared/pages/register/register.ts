import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { RegisterPayload } from '../../../core/auth/auth.models';
import { passwordsProblem } from '../../../core/utils/login-input';

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
  passwordRepeat = signal('');
  showPassword = signal(false);
  acceptedRules = signal(false);
  iin = signal(this.route.snapshot.queryParamMap.get('iin') ?? '');
  /** IIN came from the login screen — don't let it be edited here. */
  iinLocked = /^\d{12}$/.test(this.iin());

  // State
  loading = signal(false);
  errorMessage = signal('');

  // Состояние модального окна с регламентом
  showTerms = signal(false);

  // ЖК ID из query параметров (например, /auth/register?jk=UUID)
  jkId = signal(this.route.snapshot.queryParamMap.get('jk') ?? '');

  register(): void {
    if (!this.fullName().trim() || !this.phone().trim()) {
      this.errorMessage.set('Заполните ФИО и номер телефона');
      return;
    }

    const pwProblem = passwordsProblem(this.password(), this.passwordRepeat());
    if (pwProblem) {
      this.errorMessage.set(pwProblem);
      return;
    }

    if (!/^\d{12}$/.test(this.iin())) {
      this.errorMessage.set('ИИН должен содержать 12 цифр');
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
      iin: this.iin(),
    };

    this.authService.register(payload).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/app']);
      },
      error: (err) => {
        this.loading.set(false);
        if (err.status === 409 && /ИИН/.test(err.error?.error ?? '')) {
          // Already registered → back to login with the IIN filled in.
          this.router.navigate(['/login'], { queryParams: { login: this.iin() } });
          return;
        }
        this.errorMessage.set(err.error?.error || 'Ошибка регистрации. Попробуйте позже.');
      },
    });
  }
}
