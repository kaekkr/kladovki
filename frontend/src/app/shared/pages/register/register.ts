import { Component, ElementRef, computed, effect, inject, signal, viewChild } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { OwnerType, RegisterPayload } from '../../../core/auth/auth.models';
import { passwordsProblem } from '../../../core/utils/login-input';
import { normalizePhone, phoneProblem } from '../../../core/utils/phone';
import { verificationProblem } from '../../../core/utils/resident-display';
import {
  CADASTRAL_EXAMPLE,
  CADASTRAL_HINT,
  CADASTRAL_PROBLEM,
  parsePremisesCadastral,
  premisesFields,
  premisesPlaceLabel,
} from '../../../core/utils/cadastral';
import { LegalService } from '../../../core/services/legal';
import { ConsentCheckbox } from '../../components/consent-checkbox/consent-checkbox';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [FormsModule, RouterLink, ConsentCheckbox],
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

  // Property in this ЖК — only owners may rent; the ОСИ checks it.
  ownerType = signal<OwnerType>('person');
  /** Premises cadastral number: the server finds the ЖК from it (no QR needed). */
  cadastral = signal('');
  premises = computed(() => parsePremisesCadastral(this.cadastral()));
  readonly cadastralExample = CADASTRAL_EXAMPLE;
  readonly cadastralHint = CADASTRAL_HINT;
  placeLabel = premisesPlaceLabel;
  companyBin = signal('');
  companyName = signal('');
  iin = signal(this.route.snapshot.queryParamMap.get('iin') ?? '');
  /** IIN came from the login screen — don't let it be edited here. */
  iinLocked = /^\d{12}$/.test(this.iin());

  // State
  loading = signal(false);
  errorMessage = signal('');

  /** The error sits next to the submit button; bring it into view on a phone. */
  private errorBox = viewChild<ElementRef<HTMLElement>>('errorBox');
  private revealError = effect(() => {
    const el = this.errorBox()?.nativeElement;
    if (!this.errorMessage() || !el) return;
    el.scrollIntoView?.({ block: 'center', behavior: 'smooth' });
    el.focus({ preventScroll: true });
  });

  /** Version of the agreement shown in the checkbox — sent as proof of consent. */
  private termsVersion = toSignal(inject(LegalService).version('resident_terms'), { initialValue: '' });

  // ЖК ID из query параметров (например, /auth/register?jk=UUID)
  jkId = signal(this.route.snapshot.queryParamMap.get('jk') ?? '');

  setOwnerType(t: OwnerType): void {
    this.ownerType.set(t);
  }

  private property() {
    const company = this.ownerType() === 'company';
    const premises = this.premises();
    return {
      owner_type: this.ownerType(),
      ...(premises ? premisesFields(premises) : { premises_kind: 'flat' as const, premises_number: '' }),
      company_bin: company ? this.companyBin().trim() : undefined,
      company_name: company ? this.companyName().trim() : undefined,
    };
  }

  register(): void {
    if (!this.fullName().trim()) {
      this.errorMessage.set('Заполните ФИО');
      return;
    }

    const phoneIssue = phoneProblem(this.phone());
    if (phoneIssue) {
      this.errorMessage.set(phoneIssue);
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

    if (!this.premises()) {
      this.errorMessage.set(CADASTRAL_PROBLEM);
      return;
    }
    const property = this.property();
    const propertyProblem = verificationProblem(property);
    if (propertyProblem) {
      this.errorMessage.set(propertyProblem);
      return;
    }

    if (!this.acceptedRules()) {
      this.errorMessage.set('Необходимо принять Пользовательское соглашение');
      return;
    }

    this.loading.set(true);
    this.errorMessage.set('');

    const payload: RegisterPayload = {
      full_name: this.fullName().trim(),
      phone: normalizePhone(this.phone()) ?? this.phone(),
      password: this.password(),
      jk_id: this.jkId() || undefined,
      iin: this.iin(),
      ...property,
      accepted_terms_version: this.termsVersion(),
    };

    this.authService.register(payload).subscribe({
      next: () => {
        this.loading.set(false);
        // New accounts wait for the ОСИ; the app shows the status banner.
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
