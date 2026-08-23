import { Component, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

export interface AdminBulkStorageData {
  startNum: number;
  count: number;
  floor: number;
  entrance: number;
  area: number;
}

@Component({
  selector: 'app-admin-bulk-storage-modal',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './bulk_storage_modal.html',
})
export class AdminBulkStorageModal {
  close = output<void>();
  created = output<AdminBulkStorageData>();

  startNum = signal<number>(101);
  count = signal<number>(10);
  floor = signal<number>(1);
  entrance = signal<number>(1);
  area = signal<number>(3.5);

  submit(): void {
    this.created.emit({
      startNum: this.startNum(),
      count: this.count(),
      floor: this.floor(),
      entrance: this.entrance(),
      area: this.area(),
    });
    this.close.emit();
  }
}
