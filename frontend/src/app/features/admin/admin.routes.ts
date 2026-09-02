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
      {
        path: 'payments',
        loadComponent: () => import('./pages/payments/payments').then((m) => m.AdminPayments),
      },
      {
        path: 'charges',
        loadComponent: () => import('./pages/charges/charges').then((m) => m.AdminCharges),
      },
      {
        path: 'debts',
        loadComponent: () => import('./pages/debts/debts').then((m) => m.AdminDebts),
      },
      {
        path: 'documents',
        loadComponent: () => import('./pages/documents/documents').then((m) => m.AdminDocuments),
      },
      {
        path: 'settings',
        loadComponent: () => import('./pages/settings/settings').then((m) => m.AdminSettingsPage),
      },
    ],
  },
];
