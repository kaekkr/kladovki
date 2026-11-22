import { Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AuthService } from '../../../../core/auth/auth.service';
import { OwnerType } from '../../../../core/auth/auth.models';
import { ResidentService } from '../../../../core/services/resident';
import { propertyLabel, verificationProblem } from '../../../../core/utils/resident-display';
import {
  CADASTRAL_EXAMPLE,
  CADASTRAL_PROBLEM,
  parsePremisesCadastral,
  premisesFields,
  premisesPlaceLabel,
} from '../../../../core/utils/cadastral';

/** Banner for residents the ОСИ has not confirmed yet (pending / rejected / blocked). */
@Component({
  selector: 'app-account-status',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './account-status.html',
})
export class AccountStatus {
  private auth = inject(AuthService);
  private residents = inject(ResidentService);

  user = this.auth.currentUser;
  status = computed(() => {
    const u = this.user();
    return u?.role === 'resident' ? (u.status ?? 'active') : 'active';
  });
  property = computed(() => {
    const u = this.user();
    return u?.premises_number
      ? propertyLabel({
          owner_type: u.owner_type ?? 'person',
          premises_kind: u.premises_kind ?? 'flat',
          premises_number: u.premises_number,
          company_bin: u.company_bin,
          company_name: u.company_name,
        })
      : '';
  });

  // Resubmit form (rejected users)
  editing = signal(false);
  ownerType = signal<OwnerType>('person');
  cadastral = signal('');
  premises = computed(() => parsePremisesCadastral(this.cadastral()));
  readonly cadastralExample = CADASTRAL_EXAMPLE;
  placeLabel = premisesPlaceLabel;
  companyBin = signal('');
  companyName = signal('');
  saving = signal(false);
  error = signal<string | null>(null);

  startEdit(): void {
    const u = this.user();
    this.ownerType.set(u?.owner_type ?? 'person');
    this.cadastral.set(u?.premises_cadastral ?? '');
    this.companyBin.set(u?.company_bin ?? '');
    this.companyName.set(u?.company_name ?? '');
    this.error.set(null);
    this.editing.set(true);
  }

  resubmit(): void {
    const company = this.ownerType() === 'company';
    const premises = this.premises();
    if (!premises) {
      this.error.set(CADASTRAL_PROBLEM);
      return;
    }
    const v = {
      owner_type: this.ownerType(),
      ...premisesFields(premises),
      company_bin: company ? this.companyBin().trim() : '',
      company_name: company ? this.companyName().trim() : '',
    };
    const problem = verificationProblem(v);
    if (problem) {
      this.error.set(problem);
      return;
    }
    this.saving.set(true);
    this.residents.resubmit(v).subscribe({
      next: () => {
        this.saving.set(false);
        this.editing.set(false);
        this.auth.getMe().subscribe();
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err?.error?.error || 'Не удалось отправить заявку');
      },
    });
  }
}
