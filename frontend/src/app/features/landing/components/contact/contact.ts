import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { LeadService } from '../../../../core/services/lead';

@Component({
  selector: 'app-contact',
  standalone: true,
  imports: [ReactiveFormsModule],
  templateUrl: './contact.html',
})
export class Contact {
  private fb = inject(FormBuilder);
  private leadService = inject(LeadService);

  loading = signal(false);
  success = signal(false);
  error = signal<string | null>(null);

  form = this.fb.nonNullable.group({
    full_name: ['', Validators.required],
    phone: ['', Validators.required],
    email: [''],
  });

  submit() {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.loading.set(true);
    this.error.set(null);
    this.success.set(false);

    this.leadService.submit(this.form.getRawValue()).subscribe({
      next: () => {
        this.success.set(true);
        this.form.reset();
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err?.error?.error || 'Ошибка при отправке. Попробуйте позже.');
        this.loading.set(false);
      },
    });
  }
}
