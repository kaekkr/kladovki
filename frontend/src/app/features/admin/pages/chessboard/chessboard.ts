import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { StorageService } from '../../../../core/services/storage';
import { AuthService } from '../../../../core/auth/auth.service';
import { StorageUnit } from '../../../../core/models/storage';

import { AdminStorageStats } from './components/storage_stats/storage_stats';
import { AdminStorageDrawer } from './components/storage_drawer/storage_drawer';
import {
  AdminCreateStorageFormValue,
  AdminCreateStorageModal,
} from './components/create_storage_modal/create_storage_modal';
import { AdminStorageChessboard } from './components/storage_chessboard/storage_chessboard';
import { AdminBulkStorageModal } from './components/bulk_storage_modal/bulk_storage_modal';

@Component({
  selector: 'app-chessboard-page',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    AdminStorageStats,
    AdminStorageDrawer,
    AdminCreateStorageModal,
    AdminBulkStorageModal,
    AdminStorageChessboard,
  ],
  templateUrl: './chessboard.html',
})
export class AdminChessboard implements OnInit {
  private storageService = inject(StorageService);
  private authService = inject(AuthService);

  units = this.storageService.storages;

  selectedFloor = signal<number | 'all'>('all');
  selectedStatus = signal<string>('all');
  searchQuery = signal<string>('');
  selectedUnit = signal<StorageUnit | null>(null);

  isCreateModalOpen = signal<boolean>(false);
  isBulkModalOpen = signal<boolean>(false);

  floors = computed(() => {
    const list = this.units();
    if (!list.length) return [];
    const uniqueFloors = Array.from(new Set(list.map((u) => u.floor)));
    return uniqueFloors.sort((a, b) => b - a);
  });

  ngOnInit(): void {
    const user = this.authService.currentUser();
    if (user?.jk_id) {
      this.storageService.getStoragesByJK(user.jk_id).subscribe();
    }
  }

  filteredUnits = computed(() => {
    const floor = this.selectedFloor();
    const status = this.selectedStatus();
    const query = this.searchQuery().toLowerCase();

    return this.units().filter((u) => {
      const matchesFloor = floor === 'all' || u.floor === floor;
      const matchesStatus = status === 'all' || u.status === status;
      const matchesSearch = u.number.toLowerCase().includes(query);
      return matchesFloor && matchesStatus && matchesSearch;
    });
  });

  stats = computed(() => {
    const floor = this.selectedFloor();
    const baseList = floor === 'all' ? this.units() : this.units().filter((u) => u.floor === floor);

    return {
      total: baseList.length,
      free: baseList.filter((u) => u.status === 'free').length,
      occupied: baseList.filter((u) => u.status === 'occupied').length,
      locked: baseList.filter((u) => u.status === 'locked').length,
    };
  });

  selectFloor(floor: number | 'all') {
    this.selectedFloor.set(floor);
    this.closeDetails();
  }

  closeDetails() {
    this.selectedUnit.set(null);
  }

  onStorageCreated(data: AdminCreateStorageFormValue): void {
    const user = this.authService.currentUser();
    if (!user?.jk_id) return;

    this.storageService.createStorage({ ...data, jk_id: user.jk_id }).subscribe();
  }

  onBulkCreated(data: {
    startNum: number;
    count: number;
    floor: number;
    entrance: number;
    area: number;
  }): void {
    const user = this.authService.currentUser();
    if (!user?.jk_id) return;

    this.storageService.bulkCreateStorage({ ...data, jk_id: user.jk_id }).subscribe();
  }
}
