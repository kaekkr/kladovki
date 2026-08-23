import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { StorageStatus, StorageUnit } from '../../../../../../core/models/storage';

@Component({
  selector: 'app-admin-storage-chessboard',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './storage_chessboard.html',
})
export class AdminStorageChessboard {
  floors = input.required<number[]>();
  // Accept both 'all' string literal and number
  selectedFloor = input.required<number | 'all'>();
  selectedStatus = input.required<string>();
  searchQuery = input.required<string>();
  units = input.required<StorageUnit[]>();

  // Output must also allow 'all' | number
  floorChange = output<number | 'all'>();
  statusChange = output<string>();
  queryChange = output<string>();
  unitSelect = output<StorageUnit>();

  getStatusName(status: StorageStatus): string {
    switch (status) {
      case 'free':
        return 'Свободно';
      case 'occupied':
        return 'Занято';
      case 'locked':
        return 'Заблокировано';
    }
  }

  getStatusBadgeClass(status: StorageStatus): string {
    switch (status) {
      case 'free':
        return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
      case 'occupied':
        return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
      case 'locked':
        return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
    }
  }
}
