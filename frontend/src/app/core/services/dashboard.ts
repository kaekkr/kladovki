import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

// import { environment } from '../../../environments/environment.development';
import { environment } from '../../../environments/environment';
import {
  DashboardActivity,
  DashboardAttention,
  DashboardOccupancy,
  DashboardPeriod,
  DashboardRevenue,
  DashboardStats,
} from '../models/dashboard';

@Injectable({
  providedIn: 'root',
})
export class DashboardService {
  private http = inject(HttpClient);

  private baseUrl = environment.apiUrl;

  getStats(period: DashboardPeriod) {
    return this.http.get<DashboardStats>(`${this.baseUrl}/dashboard/stats`, {
      params: {
        period,
      },
    });
  }

  getRevenue(period: DashboardPeriod) {
    return this.http.get<DashboardRevenue>(`${this.baseUrl}/dashboard/revenue`, {
      params: {
        period,
      },
    });
  }

  getAttention() {
    return this.http.get<DashboardAttention[]>(`${this.baseUrl}/dashboard/attention`);
  }

  getActivity() {
    return this.http.get<DashboardActivity[]>(`${this.baseUrl}/dashboard/activity`);
  }

  getOccupancy() {
    return this.http.get<DashboardOccupancy>(`${this.baseUrl}/dashboard/occupancy`);
  }
}
