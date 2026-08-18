import { inject } from '@angular/core';
import { Router, CanActivateFn, ActivatedRouteSnapshot } from '@angular/router';
import { AuthService } from './auth.service';

export const authGuard: CanActivateFn = (route: ActivatedRouteSnapshot) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const user = auth.currentUser();

  if (!user) {
    router.navigate(['/login']);
    return false;
  }

  // Check if route requires a specific role (e.g. data: { role: 'admin' })
  const requiredRole = route.data['role'];
  if (requiredRole && user.role !== requiredRole) {
    // Redirect residents attempting to access admin areas to their client portal
    router.navigate([user.role === 'admin' ? '/admin' : '/client']);
    return false;
  }

  return true;
};
