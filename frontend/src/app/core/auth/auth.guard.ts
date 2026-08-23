import { inject } from '@angular/core';
import { Router, CanActivateFn, ActivatedRouteSnapshot } from '@angular/router';
import { AuthService } from './auth.service';
import { map, catchError, of } from 'rxjs';

export const authGuard: CanActivateFn = (route: ActivatedRouteSnapshot) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const checkRoleAndGrant = (userRole: string): boolean => {
    const requiredRole = route.data['role'];
    if (requiredRole && userRole !== requiredRole) {
      router.navigate([userRole === 'admin' ? '/admin' : '/client']);
      return false;
    }
    return true;
  };

  const currentUser = auth.currentUser();

  if (currentUser) {
    return checkRoleAndGrant(currentUser.role);
  }

  return auth.getMe().pipe(
    map((user) => {
      if (!user) {
        router.navigate(['/login']);
        return false;
      }
      return checkRoleAndGrant(user.role);
    }),
    catchError(() => {
      router.navigate(['/login']);
      return of(false);
    }),
  );
};
