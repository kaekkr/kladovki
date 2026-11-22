import { Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { homeFor, safeReturnUrl } from '../../../core/auth/return-url';

import { AuthService } from '../../../core/auth/auth.service';
import { classifyLogin } from '../../../core/utils/login-input';
import { isNativeApp } from '../../../core/auth/native-start.guard';

/**
 * Step 1: resident enters IIN (OSI admin — email). Step 2: password.
 * An IIN without an account goes to the first-login screen.
 */
@Component({
  selector: 'app-login',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  templateUrl: './login.html',
})
export class Login {
  private fb = inject(FormBuilder);
  private auth = inject(AuthService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);

  errorMessage = signal<string | null>(null);
  isLoading = signal<boolean>(false);
  showPassword = signal<boolean>(false);
  step = signal<'identify' | 'password'>('identify');
  /** Android app: no links to the marketing site. */
  readonly native = isNativeApp();

  /** JK from the OSI's QR link, carried to first-login registration. */
  private jk = this.route.snapshot.queryParamMap.get('jk') ?? '';

  form = this.fb.nonNullable.group({
    login: [this.route.snapshot.queryParamMap.get('login') ?? '', [Validators.required]],
    password: [''],
  });

  /** Set by the HTTP interceptor when a 401 sent the user back here. */
  readonly sessionExpired = this.route.snapshot.queryParamMap.get('expired') === '1';

  private loginValue = toSignal(this.form.controls.login.valueChanges, {
    initialValue: this.form.controls.login.value,
  });
  /** Only residents (ИИН) get the «обратитесь в ОСИ» hint; the ОСИ themselves don't. */
  isResidentLogin = computed(() => classifyLogin(this.loginValue()) === 'iin');

  constructor() {
    if (this.form.controls.login.value) {
      this.step.set('password');
    }
  }

  /** Typing a different login goes back to step 1 (the IIN must be checked again). */
  onLoginEdited(): void {
    if (this.step() !== 'password') return;
    this.step.set('identify');
    this.form.controls.password.setValue('');
    this.errorMessage.set(null);
  }

  onSubmit(): void {
    this.errorMessage.set(null);
    if (this.step() === 'identify') {
      this.identify();
    } else {
      this.signIn();
    }
  }

  private identify(): void {
    const login = this.form.controls.login.value.trim();
    const kind = classifyLogin(login);
    if (kind === 'invalid') {
      this.errorMessage.set('Введите ИИН (12 цифр), телефон или email');
      return;
    }
    if (kind !== 'iin') {
      this.step.set('password');
      return;
    }

    this.isLoading.set(true);
    this.auth.iinStatus(login).subscribe({
      next: ({ registered }) => {
        this.isLoading.set(false);
        if (registered) {
          this.step.set('password');
          return;
        }
        this.router.navigate(['/auth/register'], {
          queryParams: { iin: login, ...(this.jk ? { jk: this.jk } : {}) },
        });
      },
      error: (err) => {
        this.isLoading.set(false);
        this.errorMessage.set(err.error?.error || 'Сервер временно недоступен. Попробуйте позже.');
      },
    });
  }

  private signIn(): void {
    const { login, password } = this.form.getRawValue();
    if (password.length < 6) {
      this.errorMessage.set('Введите пароль (минимум 6 символов)');
      return;
    }

    this.isLoading.set(true);
    this.auth.login({ login: login.trim(), password }).subscribe({
      next: (user) => {
        this.isLoading.set(false);
        // Back to the page that asked for login (e.g. a link from a notification).
        const back = safeReturnUrl(this.route.snapshot.queryParamMap.get('returnUrl'), user.role);
        this.router.navigateByUrl(back ?? homeFor(user.role));
      },
      error: (err) => {
        this.isLoading.set(false);
        this.errorMessage.set(err.error?.error || 'Неверный логин или пароль.');
      },
    });
  }
}
