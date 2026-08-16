import { Routes } from '@angular/router';
import { authGuard } from '../../core/auth/auth.guard';

export const ADMIN_ROUTES: Routes = [
  {
    path: 'login',
    loadComponent: () => import('./pages/login/login').then((m) => m.AdminLogin),
  },
  {
    path: '',
    canActivate: [authGuard],
    loadComponent: () => import('./admin').then((m) => m.Admin),
    children: [
      {
        path: '',
        redirectTo: 'dashboard',
        pathMatch: 'full',
      },
      // {
      //   path: 'dashboard',
      //   loadComponent: () => import('./pages/dashboard/dashboard').then((m) => m.AdminDashboard),
      // },
    ],
  },
];
