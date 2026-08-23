import { Component } from '@angular/core';
import { AdminStats } from './components/stats/stats';
import { AdminRevenueChart } from './components/revenue_chart/revenue_chart';
import { AdminAttention } from './components/attention/attention';
import { AdminRecentActivity } from './components/recent_activity/recent_activity';
import { AdminOccupancySummary } from './components/occupancy_summary/occupancy_summary';

@Component({
  selector: 'app-admin-dashboard',
  standalone: true,
  imports: [
    AdminStats,
    AdminRevenueChart,
    AdminAttention,
    AdminRecentActivity,
    AdminOccupancySummary,
  ],
  templateUrl: './dashboard.html',
})
export class AdminDashboard { }
