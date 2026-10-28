import { Component, EventEmitter, Output, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Storage } from '../../../../../../core/models/storage';

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

    if (!this.number.trim()) {
      this.validationError.set('Введите номер кладовой.');
      return;
    }

    if (this.area === null || this.area <= 0) {
      this.validationError.set('Площадь должна быть больше 0.');
      return;
    }

    if (this.floor === null) {
      this.validationError.set('Введите этаж.');
      return;
    }

    if (this.entrance === null) {
      this.validationError.set('Введите подъезд.');
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
