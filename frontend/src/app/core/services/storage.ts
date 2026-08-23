import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap, catchError, of } from 'rxjs';
import { environment } from '../../../environments/environment.development';
import { StorageUnit, CreateStoragePayload, BulkCreateStoragePayload } from '../models/storage';

@Injectable({ providedIn: 'root' })
export class StorageService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  storages = signal<StorageUnit[]>([]);

  getStoragesByJK(jkId: string) {
    return this.http.get<StorageUnit[]>(`${this.baseUrl}/storages/jk/${jkId}`).pipe(
      tap((data) => this.storages.set(data || [])),
      catchError(() => {
        this.storages.set([]);
        return of([]);
      }),
    );
  }

  createStorage(payload: CreateStoragePayload) {
    return this.http.post<StorageUnit>(`${this.baseUrl}/storages`, payload).pipe(
      tap((newStorage) => {
        this.storages.update((list) => [...list, newStorage]);
      }),
    );
  }

  bulkCreateStorage(payload: BulkCreateStoragePayload) {
    return this.http.post<StorageUnit[]>(`${this.baseUrl}/storages/bulk`, payload).pipe(
      tap((newStorages) => {
        this.storages.update((list) => [...list, ...newStorages]);
      }),
    );
  }

  updateStorage(id: string, payload: Partial<StorageUnit>) {
    return this.http.put<StorageUnit>(`${this.baseUrl}/storages/${id}`, payload).pipe(
      tap((updated) => {
        this.storages.update((list) => list.map((item) => (item.id === id ? updated : item)));
      }),
    );
  }

  deleteStorage(id: string) {
    return this.http.delete(`${this.baseUrl}/storages/${id}`).pipe(
      tap(() => {
        this.storages.update((list) => list.filter((item) => item.id !== id));
      }),
    );
  }
}
