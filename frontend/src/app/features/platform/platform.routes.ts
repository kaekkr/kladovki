import { Routes } from '@angular/router';

/** Super-admin panel: the platform operator creates ЖК and their ОСИ accounts. */
export const PLATFORM_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () => import('./layout/platform-layout').then((m) => m.PlatformLayout),
    children: [
      {
        path: '',
        loadComponent: () => import('./pages/jk-list').then((m) => m.PlatformJKList),
      },
      {
        path: 'jks/new',
        loadComponent: () => import('./pages/jk-create').then((m) => m.PlatformJKCreate),
      },
      {
        path: 'jks/:id',
        loadComponent: () => import('./pages/jk-details').then((m) => m.PlatformJKDetails),
      },
    ],
  },
];
