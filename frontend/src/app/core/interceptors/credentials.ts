import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { TimeoutError, catchError, throwError, timeout } from 'rxjs';

import { AuthService } from '../auth/auth.service';
import { attachesToken, tokenStore } from '../auth/token-session';
import { environment } from '../../../environments/environment';

/** Endpoints where 401 is an expected answer, not an expired session. */
const AUTH_CALLS = [
  '/auth/login',
  '/auth/me',
  '/auth/register',
  '/auth/iin-status',
  '/auth/logout',
];

/**
 * A request that hangs (server waking up, the app was in the background) must
 * end in an error the page can show with a «Повторить» button — otherwise the
 * loading skeleton spins forever.
 */
export const REQUEST_TIMEOUT_MS = 30_000;

/** Same shape as a server error, so pages show `err.error.error` as usual. */
function timeoutError(url: string): HttpErrorResponse {
  return new HttpErrorResponse({
    url,
    status: 0,
    statusText: 'Timeout',
    error: { error: 'Сервер не ответил вовремя. Проверьте интернет и попробуйте ещё раз.' },
  });
}

/**
 * Sends the session cookie with every request. When the session has expired
 * (401 from any other endpoint) the user is sent to login and comes back to
 * the same page afterwards.
 */
export const credentialsInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  // Website: HttpOnly cookie. Mobile app: the cookie never comes back cross-site,
  // so the stored token is sent instead (only to our API).
  const token = tokenStore.get();
  const authed =
    token && attachesToken(req.url, environment.apiUrl)
      ? req.clone({ withCredentials: true, setHeaders: { Authorization: `Bearer ${token}` } })
      : req.clone({ withCredentials: true });

  return next(authed).pipe(
    timeout(REQUEST_TIMEOUT_MS),
    catchError((err: unknown) => {
      if (err instanceof TimeoutError) {
        return throwError(() => timeoutError(req.url));
      }
      if (
        err instanceof HttpErrorResponse &&
        err.status === 401 &&
        !AUTH_CALLS.some((p) => req.url.includes(p)) &&
        auth.currentUser()
      ) {
        // The login page says «Сессия истекла» and keeps the resident's ИИН filled in.
        const iin = auth.currentUser()?.iin;
        auth.currentUser.set(null);
        tokenStore.clear();
        router.navigate(['/login'], {
          queryParams: { returnUrl: router.url, expired: 1, ...(iin ? { login: iin } : {}) },
        });
      }
      return throwError(() => err);
    }),
  );
};
