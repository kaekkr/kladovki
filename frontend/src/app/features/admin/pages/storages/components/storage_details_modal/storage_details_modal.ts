import { Component, EventEmitter, Input, Output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

import { Storage } from '../../../../../../core/models/storage';
import { storageFormError } from '../../../../../../core/utils/storage-form';

@Component({
  selector: 'app-admin-storage-details-modal',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './storage_details_modal.html',
})
export class AdminStorageDetailsModal {
  private current: Storage | null = null;

  /** A new object after a successful save ends the edit mode; an error keeps it open. */
  @Input() set storage(value: Storage | null) {
    if (value !== this.current) {
      this.editing.set(false);
      this.saving.set(false);
    }
    this.current = value;
  }
  get storage(): Storage | null {
    return this.current;
  }

  /** Server error of the last save, shown inside the modal. */
  @Input() set error(value: string | null) {
    this.errorText.set(value);
    if (value) this.saving.set(false);
  }
  errorText = signal<string | null>(null);

  /** Number, area, floor and entrance can be changed only while the room is free (server rule). */
  get editable(): boolean {
    return this.current?.status === 'free';
  }

  @Output() closed = new EventEmitter<void>();
  @Output() saved = new EventEmitter<Storage>();
  @Output() deleteRequested = new EventEmitter<Storage>();

  editing = signal(false);

  editNumber = '';
  editArea = 0;
  editFloor = 0;
  editEntrance = 0;

  saving = signal(false);
  validationError = signal<string | null>(null);

  startEdit(): void {
    if (!this.storage) {
      return;
    }

    this.editNumber = this.storage.number;
    this.editArea = this.storage.area;
    this.editFloor = this.storage.floor;
    this.editEntrance = this.storage.entrance;

    this.editing.set(true);
  }

  cancelEdit(): void {
    this.validationError.set(null);
    this.editing.set(false);
  }

  save(): void {
    if (!this.storage) {
      return;
    }
    const error = storageFormError({
      number: this.editNumber,
      area: this.editArea,
      floor: this.editFloor,
      entrance: this.editEntrance,
    });
    this.validationError.set(error);
    if (error) {
      return;
    }

    this.saving.set(true);
    this.errorText.set(null);

    this.saved.emit({
      ...this.storage,
      number: this.editNumber,
      area: this.editArea,
      floor: this.editFloor,
      entrance: this.editEntrance,
    });
  }

  close(): void {
    if (this.saving()) {
      return;
    }

    this.closed.emit();
  }
}
