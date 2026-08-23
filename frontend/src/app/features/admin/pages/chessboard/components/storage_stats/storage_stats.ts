import { Component, input, output } from '@angular/core';

@Component({
  selector: 'app-admin-storage-stats',
  standalone: true,
  templateUrl: './storage_stats.html',
})
export class AdminStorageStats {
  stats = input.required<{ total: number; free: number; occupied: number; locked: number }>();
  selectedStatus = input.required<string>();
  statusChange = output<string>();
}
