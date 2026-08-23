export type StorageStatus = 'free' | 'occupied' | 'locked';

export interface StorageUnit {
  id: string;
  jk_id: string;
  number: string;
  area: number;
  floor: number;
  entrance: number;
  status: StorageStatus;
  created_at: string;
}

export interface CreateStoragePayload {
  jk_id?: string;
  number: string;
  area: number;
  floor: number;
  entrance: number;
}

export interface BulkCreateStoragePayload {
  jk_id: string;
  startNum: number;
  count: number;
  floor: number;
  entrance: number;
  area: number;
}
