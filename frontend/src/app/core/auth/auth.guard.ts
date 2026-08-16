import { inject } from '@angular/core';
import { Router, CanActivateFn } from '@angular/router';
import { AuthService } from './auth.service';

export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const user = auth.currentUser();

  if (user && (user.role === 'admin' || user.role === 'superadmin')) {
    return true;
  }

  router.navigate(['/admin/login']);
  return false;
};
