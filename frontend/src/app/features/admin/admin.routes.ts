import { Routes } from '@angular/router';

export const ADMIN_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () => import('./admin').then((m) => m.Admin),
    children: [
      {
        path: '',
        redirectTo: 'dashboard',
        pathMatch: 'full',
      },
      {
        path: 'dashboard',
        loadComponent: () => import('./pages/dashboard/dashboard').then((m) => m.AdminDashboard),
      },
      {
        path: 'storages',
        loadComponent: () => import('./pages/storages/storages').then((m) => m.AdminStorages),
      },
      {
        path: 'rentals',
        loadComponent: () => import('./pages/rentals/rentals').then((m) => m.AdminRentals),
      },
    ],
  },
];
