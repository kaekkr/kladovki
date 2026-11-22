import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';

import { JKForm, PlatformAdmin, PlatformJK } from '../../../core/models/platform';
import { PlatformService } from '../../../core/services/platform';
import { apiErrorText, validateJKForm } from '../../../core/utils/platform-form';
import { JKAdmins } from '../components/jk-admins';
import { JKFields } from '../components/jk-fields';

const toForm = (jk: PlatformJK): JKForm => ({
  name: jk.name,
  address: jk.address,
  cadastral_number: jk.cadastral_number,
});

/** One ЖК: edit its details and manage its ОСИ accounts. */
@Component({
  selector: 'app-platform-jk-details',
  standalone: true,
  imports: [RouterLink, JKFields, JKAdmins],
  template: `
    <div class="max-w-2xl space-y-6">
      <a routerLink="/platform" class="text-xs text-brand-muted hover:text-brand-text">← Все ЖК</a>

      @if (loading()) {
        <div class="h-40 animate-pulse rounded-lg border border-brand-line bg-brand-surface"></div>
        <div class="h-32 animate-pulse rounded-lg border border-brand-line bg-brand-surface"></div>
      } @else if (loadError()) {
        <div
          class="rounded-lg border border-brand-line bg-brand-surface p-5 text-sm text-status-overdue"
        >
          {{ loadError() }}
          <button
            type="button"
            (click)="load()"
            class="ml-2 rounded-md border border-brand-line px-3 py-1 text-xs font-medium text-brand-text hover:border-brand-accent"
          >
            Повторить
          </button>
        </div>
      } @else if (jk(); as current) {
        <div>
          <h1 class="text-2xl font-semibold tracking-tight text-brand-text break-words">
            {{ current.name }}
          </h1>
          <p class="mt-1 text-sm text-brand-muted">
            Кладовок: {{ current.storages_count }} · ОСИ: {{ current.admins_count }} · Жителей:
            {{ current.residents_count }}
          </p>
        </div>

        <form
          class="overflow-hidden rounded-lg border border-brand-line bg-brand-surface"
          (submit)="$event.preventDefault(); save()"
          novalidate
        >
          <div class="border-b border-brand-line px-5 py-4">
            <h2 class="text-sm font-medium text-brand-text">Данные ЖК</h2>
          </div>
          <div class="space-y-4 px-5 py-4">
            <app-jk-fields [(form)]="form" />
            @if (saveError()) {
              <div
                class="rounded-md border border-brand-line px-3 py-2 text-sm text-status-overdue"
                role="alert"
              >
                {{ saveError() }}
              </div>
            }
            @if (saved()) {
              <div class="rounded-md border border-brand-line px-3 py-2 text-sm text-brand-text">
                Сохранено
              </div>
            }
            <button
              type="submit"
              [disabled]="saving()"
              class="w-full rounded-md bg-brand-accent px-4 py-2 text-sm font-semibold text-brand-bg hover:opacity-90 disabled:opacity-50 sm:w-auto"
            >
              {{ saving() ? 'Сохраняем…' : 'Сохранить' }}
            </button>
          </div>
        </form>

        <app-jk-admins [jkId]="current.id" [admins]="admins()" (changed)="reloadAdmins()" />
      }
    </div>
  `,
})
export class PlatformJKDetails implements OnInit {
  private platform = inject(PlatformService);

  private route = inject(ActivatedRoute);

  /** The ЖК id from /platform/jks/:id. */
  private id = (): string => this.route.snapshot.paramMap.get('id') ?? '';

  jk = signal<PlatformJK | null>(null);
  admins = signal<PlatformAdmin[]>([]);
  form = signal<JKForm>({ name: '', address: '', cadastral_number: '' });
  loading = signal(true);
  loadError = signal<string | null>(null);
  saving = signal(false);
  saveError = signal<string | null>(null);
  saved = signal(false);

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.loadError.set(null);
    forkJoin({
      jk: this.platform.getJK(this.id()),
      admins: this.platform.listAdmins(this.id()),
    }).subscribe({
      next: ({ jk, admins }) => {
        this.jk.set(jk);
        this.form.set(toForm(jk));
        this.admins.set(admins);
        this.loading.set(false);
      },
      error: (err) => {
        this.loadError.set(apiErrorText(err, 'Не удалось загрузить ЖК.'));
        this.loading.set(false);
      },
    });
  }

  save(): void {
    const problem = validateJKForm(this.form());
    this.saved.set(false);
    if (problem) {
      this.saveError.set(problem);
      return;
    }
    this.saving.set(true);
    this.saveError.set(null);
    this.platform.updateJK(this.id(), this.form()).subscribe({
      next: (jk) => {
        this.jk.set(jk);
        this.form.set(toForm(jk));
        this.saving.set(false);
        this.saved.set(true);
      },
      error: (err) => {
        this.saving.set(false);
        this.saveError.set(apiErrorText(err, 'Не удалось сохранить ЖК.'));
      },
    });
  }

  reloadAdmins(): void {
    forkJoin({
      jk: this.platform.getJK(this.id()),
      admins: this.platform.listAdmins(this.id()),
    }).subscribe({
      next: ({ jk, admins }) => {
        this.jk.set({ ...jk });
        this.admins.set(admins);
      },
      error: () => this.saveError.set('Список администраторов мог устареть — обновите страницу.'),
    });
  }
}
