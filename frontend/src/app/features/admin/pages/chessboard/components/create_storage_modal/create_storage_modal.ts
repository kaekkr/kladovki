import { Component, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { StorageStatus } from '../../../../../../core/models/storage';

export interface AdminCreateStorageFormValue {
  number: string;
  floor: number;
  entrance: number;
  area: number;
  status: StorageStatus;
}

@Component({
  selector: 'app-admin-create-storage-modal',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './create_storage_modal.html',
})
export class AdminCreateStorageModal {
  close = output<void>();
  created = output<AdminCreateStorageFormValue>();

  number = signal('');
  floor = signal(1);
  entrance = signal(1);
  area = signal(3.5);
  status = signal<StorageStatus>('free');

  submit() {
    if (!this.number().trim()) return;

    this.created.emit({
      number: this.number(),
      floor: this.floor(),
      entrance: this.entrance(),
      area: this.area(),
      status: this.status(),
    });
    this.close.emit();
  }
}
