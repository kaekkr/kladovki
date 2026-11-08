import { inject } from '@angular/core';
import { Router, CanActivateFn, ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { AuthService } from './auth.service';
import { map, catchError, of } from 'rxjs';
import { homeFor } from './return-url';

/** Requires a session; wrong area → own home; no session → /login?returnUrl=… */
export const authGuard: CanActivateFn = (route: ActivatedRouteSnapshot, state: RouterStateSnapshot) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const toLogin = () => router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });

  const checkRole = (userRole: string) => {
    const requiredRole = route.data['role'];
    return requiredRole && userRole !== requiredRole ? router.parseUrl(homeFor(userRole)) : true;
  };

  const currentUser = auth.currentUser();
  if (currentUser) {
    return checkRole(currentUser.role);
  }

  return auth.getMe().pipe(
    map((user) => (user ? checkRole(user.role) : toLogin())),
    catchError(() => of(toLogin())),
  );
};
