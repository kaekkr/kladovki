import { Component, EventEmitter, Output, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Storage } from '../../../../../../core/models/storage';
import { storageFormError } from '../../../../../../core/utils/storage-form';

@Component({
  selector: 'app-admin-storage-create-modal',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './storage_create_modal.html',
})
export class AdminStorageCreateModal {
  @Output() closed = new EventEmitter<void>();
  @Output() created = new EventEmitter<Storage>();

  saving = input(false);
  serverError = input<string | null>(null);

  number = '';
  area: number | null = null;
  floor: number | null = null;
  entrance: number | null = null;

  validationError = signal<string | null>(null);

  submit(): void {
    this.validationError.set(null);

    const error = storageFormError({
      number: this.number,
      area: this.area,
      floor: this.floor,
      entrance: this.entrance,
    });
    if (error || this.area === null || this.floor === null || this.entrance === null) {
      this.validationError.set(error);
      return;
    }

    this.created.emit({
      id: '',
      jk_id: '',
      number: this.number.trim(),
      area: this.area,
      floor: this.floor,
      entrance: this.entrance,
      status: 'free',
      created_at: '',
    });
  }

  close(): void {
    if (this.saving()) return;
    this.closed.emit();
  }
}
