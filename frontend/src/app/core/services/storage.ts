import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

// import { environment } from '../../../environments/environment.development';
import { environment } from '../../../environments/environment';
import { Storage } from '../models/storage';

@Injectable({
  providedIn: 'root',
})
export class StorageService {
  private http = inject(HttpClient);
  private baseUrl = environment.apiUrl;

  create(payload: {
    jk_id: string;
    number: string;
    area: number;
    floor: number;
    entrance: number;
  }) {
    return this.http.post<Storage>(`${this.baseUrl}/storages`, payload);
  }

  listByJK(jkId: string) {
    return this.http.get<Storage[]>(`${this.baseUrl}/storages/jk/${jkId}`);
  }

  getById(id: string) {
    return this.http.get<Storage>(`${this.baseUrl}/storages/${id}`);
  }

  update(
    id: string,
    data: {
      number: string;
      area: number;
      floor: number;
      entrance: number;
    },
  ) {
    return this.http.put<Storage>(`${this.baseUrl}/storages/${id}`, data);
  }

  delete(id: string) {
    return this.http.delete<{ message: string }>(`${this.baseUrl}/storages/${id}`);
  }
}
