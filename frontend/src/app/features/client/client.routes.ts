import { Routes } from '@angular/router';

export const CLIENT_ROUTES: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () => import('./pages/dashboard/dashboard').then((m) => m.ClientDashboard),
  },
  {
    path: 'storages',
    loadComponent: () => import('./pages/storages/storages').then((m) => m.ClientStorages),
  },
  {
    path: 'storages/:id',
    loadComponent: () =>
      import('./pages/storages/storage-details/storage-details').then(
        (m) => m.ClientStorageDetails,
      ),
  },
  {
    path: 'rentals',
    loadComponent: () => import('./pages/rentals/rentals').then((m) => m.ClientRentals),
  },
  {
    path: 'rentals/:id',
    loadComponent: () =>
      import('./pages/rentals/rental-details/rental-details').then((m) => m.ClientRentalDetails),
  },
  // {
  //   path: 'payments',
  //   loadComponent: () => import('./pages/payments/payments').then((m) => m.ClientPayments),
  // },
  // {
  //   path: 'documents',
  //   loadComponent: () => import('./pages/documents/documents').then((m) => m.ClientDocuments),
  // },
  // {
  //   path: 'notifications',
  //   loadComponent: () =>
  //     import('./pages/notifications/notifications').then((m) => m.ClientNotifications),
  // },
  // {
  //   path: 'profile',
  //   loadComponent: () => import('./pages/profile/profile').then((m) => m.ClientProfile),
  // },
];
