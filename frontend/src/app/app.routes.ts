import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';

export const routes: Routes = [
  {
    path: '',
    loadChildren: () => import('./features/landing/landing.routes').then((m) => m.LANDING_ROUTES),
  },
  {
    path: 'login',
    loadComponent: () => import('./shared/pages/login/login').then((m) => m.Login),
  },
  {
    path: 'admin',
    canActivate: [authGuard],
    data: { role: 'admin' },
    loadChildren: () => import('./features/admin/admin.routes').then((m) => m.ADMIN_ROUTES),
  },
  {
    path: 'app',
    canActivate: [authGuard],
    data: { role: 'resident' },
    loadComponent: () => import('./features/client/client').then((m) => m.Client),
    children: [
      {
        path: '',
        loadChildren: () => import('./features/client/client.routes').then((m) => m.CLIENT_ROUTES),
      },
    ],
  },
  {
    path: '**',
    redirectTo: '',
  },
];
