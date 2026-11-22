import { Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { PlatformJK } from '../../../core/models/platform';
import { PlatformService } from '../../../core/services/platform';
import { apiErrorText } from '../../../core/utils/platform-form';

/** All ЖК on the platform with their counts. */
@Component({
  selector: 'app-platform-jk-list',
  standalone: true,
  imports: [RouterLink],
  template: `
    <div class="space-y-6">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight text-brand-text">Жилые комплексы</h1>
          <p class="mt-1 text-sm text-brand-muted">ЖК на платформе и их администраторы ОСИ</p>
        </div>
        <a
          routerLink="/platform/jks/new"
          class="rounded-md bg-brand-accent px-4 py-2 text-sm font-semibold text-brand-bg hover:opacity-90"
        >
          + Создать ЖК
        </a>
      </div>

      @if (loading()) {
        <div class="space-y-3">
          @for (i of [1, 2, 3]; track i) {
            <div
              class="h-20 animate-pulse rounded-lg border border-brand-line bg-brand-surface"
            ></div>
          }
        </div>
      } @else if (error()) {
        <div
          class="rounded-lg border border-brand-line bg-brand-surface p-5 text-sm text-status-overdue"
        >
          {{ error() }}
          <button
            type="button"
            (click)="load()"
            class="ml-2 rounded-md border border-brand-line px-3 py-1 text-xs font-medium text-brand-text hover:border-brand-accent"
          >
            Повторить
          </button>
        </div>
      } @else if (jks().length === 0) {
        <div
          class="rounded-lg border border-brand-line bg-brand-surface p-6 text-center text-sm text-brand-muted"
        >
          Пока нет ни одного ЖК. Создайте первый — вместе с учётной записью ОСИ.
        </div>
      } @else {
        <ul class="space-y-3">
          @for (jk of jks(); track jk.id) {
            <li>
              <a
                [routerLink]="['/platform/jks', jk.id]"
                class="block rounded-lg border border-brand-line bg-brand-surface p-4 transition-colors hover:border-brand-accent"
              >
                <div class="flex flex-wrap items-start justify-between gap-2">
                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-brand-text">{{ jk.name }}</p>
                    <p class="mt-0.5 text-xs text-brand-muted break-words">{{ jk.address }}</p>
                    <p class="mt-0.5 text-xs text-brand-subtle">
                      Кадастровый №: {{ jk.cadastral_number || 'не указан' }}
                    </p>
                  </div>
                  <i class="fas fa-chevron-right mt-1 text-xs text-brand-subtle"></i>
                </div>
                <div class="mt-3 flex flex-wrap gap-2 text-xs">
                  <span class="rounded-md bg-brand-elevated px-2 py-1 text-brand-muted">
                    Кладовок: <b class="text-brand-text">{{ jk.storages_count }}</b>
                  </span>
                  <span class="rounded-md bg-brand-elevated px-2 py-1 text-brand-muted">
                    ОСИ: <b class="text-brand-text">{{ jk.admins_count }}</b>
                  </span>
                  <span class="rounded-md bg-brand-elevated px-2 py-1 text-brand-muted">
                    Жителей: <b class="text-brand-text">{{ jk.residents_count }}</b>
                  </span>
                </div>
              </a>
            </li>
          }
        </ul>
      }
    </div>
  `,
})
export class PlatformJKList {
  private platform = inject(PlatformService);

  jks = signal<PlatformJK[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);

  constructor() {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.error.set(null);
    this.platform.listJKs().subscribe({
      next: (list) => {
        this.jks.set(list);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(apiErrorText(err, 'Не удалось загрузить список ЖК.'));
        this.loading.set(false);
      },
    });
  }
}
