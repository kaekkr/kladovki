import { Component, input, output } from '@angular/core';
import { StorageUnit } from '../../../../../../core/models/storage';

@Component({
  selector: 'app-admin-storage-drawer',
  standalone: true,
  templateUrl: './storage_drawer.html',
})
export class AdminStorageDrawer {
  unit = input.required<StorageUnit>();
  close = output<void>();

  getStatusName(status: string): string {
    return status === 'free' ? 'Свободно' : status === 'occupied' ? 'Занято' : 'Заблокировано';
  }

  getStatusBadgeClass(status: string): string {
    return status === 'free'
      ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
      : status === 'occupied'
        ? 'bg-rose-500/10 text-rose-400 border-rose-500/20'
        : 'bg-amber-500/10 text-amber-400 border-amber-500/20';
  }
}
