import { Component, computed, effect, inject, input, signal } from '@angular/core';

import { Agreement, AgreementDoc, AgreementLang } from '../../../core/models/legal';
import { LegalService } from '../../../core/services/legal';
import { agreementBlocks } from '../../../core/utils/agreement-format';

/** Renders a user agreement with a Русский / Қазақша switch. */
@Component({
  selector: 'app-agreement',
  standalone: true,
  template: `
    <div class="flex items-center justify-between gap-3">
      @if (agreement(); as a) {
        <p class="text-[10px] text-brand-subtle">Редакция от {{ a.version }}</p>
      } @else {
        <span></span>
      }
      <div class="flex overflow-hidden rounded-md border border-brand-line text-[11px]">
        @for (l of langs; track l.key) {
          <button
            type="button"
            (click)="lang.set(l.key)"
            class="px-2.5 py-1 transition-colors"
            [class]="lang() === l.key ? 'bg-brand-accent/15 text-brand-accent' : 'text-brand-muted hover:text-brand-text'"
          >
            {{ l.label }}
          </button>
        }
      </div>
    </div>

    @if (error()) {
      <p class="mt-4 text-xs text-status-overdue">Не удалось загрузить текст соглашения.</p>
    } @else if (agreement(); as a) {
      <h2 class="mt-3 text-sm font-semibold leading-snug text-brand-text">{{ a.title }}</h2>
      <div class="mt-3 space-y-2 text-xs leading-relaxed text-brand-muted">
        @for (b of blocks(); track $index) {
          @switch (b.kind) {
            @case ('heading') {
              <p class="pt-2 text-[13px] font-semibold text-brand-text">{{ b.text }}</p>
            }
            @case ('clause') {
              <p>{{ b.text }}</p>
            }
            @default {
              <p class="text-brand-subtle">{{ b.text }}</p>
            }
          }
        }
      </div>
    } @else {
      <div class="mt-4 h-40 animate-pulse rounded-md bg-brand-elevated/40"></div>
    }
  `,
})
export class AgreementView {
  private legal = inject(LegalService);

  doc = input.required<AgreementDoc>();
  lang = signal<AgreementLang>('ru');
  langs: { key: AgreementLang; label: string }[] = [
    { key: 'ru', label: 'Русский' },
    { key: 'kk', label: 'Қазақша' },
  ];

  agreement = signal<Agreement | null>(null);
  error = signal(false);
  blocks = computed(() => agreementBlocks(this.agreement()?.text ?? ''));

  constructor() {
    effect(() => {
      const doc = this.doc();
      const lang = this.lang();
      this.error.set(false);
      this.legal.get(doc, lang).subscribe({
        next: (a) => this.agreement.set(a),
        error: () => this.error.set(true),
      });
    });
  }
}
