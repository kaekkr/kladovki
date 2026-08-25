import { Component, EventEmitter, Output, signal } from '@angular/core';
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

  number = '';
  area: number | null = null;
  floor: number | null = null;
  entrance: number | null = null;

  saving = signal(false);
  error = signal<string | null>(null);

  submit(): void {
    this.error.set(null);

    if (!this.number.trim()) {
      this.error.set('Введите номер кладовой.');
      return;
    }

    if (this.area === null || this.area <= 0) {
      this.error.set('Площадь должна быть больше 0.');
      return;
    }

    if (this.floor === null) {
      this.error.set('Введите этаж.');
      return;
    }

    if (this.entrance === null) {
      this.error.set('Введите подъезд.');
      return;
    }

    this.saving.set(true);

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
    if (this.saving()) {
      return;
    }

    this.closed.emit();
  }
}
