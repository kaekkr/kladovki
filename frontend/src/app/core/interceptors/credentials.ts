import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

import { AuthService } from '../auth/auth.service';

/** Endpoints where 401 is an expected answer, not an expired session. */
const AUTH_CALLS = ['/auth/login', '/auth/me', '/auth/register', '/auth/iin-status', '/auth/logout'];

/**
 * Sends the session cookie with every request. When the session has expired
 * (401 from any other endpoint) the user is sent to login and comes back to
 * the same page afterwards.
 */
export const credentialsInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  return next(req.clone({ withCredentials: true })).pipe(
    catchError((err: unknown) => {
      if (
        err instanceof HttpErrorResponse &&
        err.status === 401 &&
        !AUTH_CALLS.some((p) => req.url.includes(p)) &&
        auth.currentUser()
      ) {
        auth.currentUser.set(null);
        router.navigate(['/login'], { queryParams: { returnUrl: router.url } });
      }
      return throwError(() => err);
    }),
  );
};
