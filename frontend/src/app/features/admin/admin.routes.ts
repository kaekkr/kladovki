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
        path: 'chessboard',
        loadComponent: () => import('./pages/chessboard/chessboard').then((m) => m.AdminChessboard),
      },
    ],
  },
];
