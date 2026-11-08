import { Component, computed, inject } from '@angular/core';
import { Location } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';

import { AgreementDoc } from '../../../core/models/legal';
import { AgreementView } from '../../components/agreement/agreement';

/** Public page with an agreement: /terms/resident_terms, /terms/osi_rules. */
@Component({
  selector: 'app-terms',
  standalone: true,
  imports: [AgreementView],
  template: `
    <div class="min-h-screen bg-brand-bg px-4 py-6">
      <div class="mx-auto w-full max-w-2xl">
        <button type="button" (click)="back()" class="mb-4 inline-flex items-center gap-2 text-xs text-brand-muted hover:text-brand-text">
          <i class="fas fa-arrow-left text-[10px]"></i> Назад
        </button>
        <div class="rounded-xl border border-brand-line bg-brand-surface p-5">
          <app-agreement [doc]="doc()" />
        </div>
      </div>
    </div>
  `,
})
export class Terms {
  private location = inject(Location);
  private params = toSignal(inject(ActivatedRoute).paramMap.pipe(map((p) => p.get('doc'))));
  doc = computed<AgreementDoc>(() => (this.params() === 'osi_rules' ? 'osi_rules' : 'resident_terms'));

  back(): void {
    this.location.back();
  }
}
