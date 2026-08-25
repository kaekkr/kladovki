export type StorageStatus = 'free' | 'occupied' | 'locked';

export interface Storage {
  id: string;
  jk_id: string;
  number: string;
  area: number;
  floor: number;
  entrance: number;
  status: StorageStatus;
  created_at: string;
}

export interface CreateStorageRequest {
  jk_id?: string;
  number: string;
  area: number;
  floor: number;
  entrance: number;
}

export interface UpdateStorageRequest {
  number: string;
  area: number;
  floor: number;
  entrance: number;
  status: StorageStatus;
}
