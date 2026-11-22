import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { Capacitor } from '@capacitor/core';
import { map } from 'rxjs';

import { AuthService } from './auth.service';
import { homeFor } from './return-url';

/** The marketing landing is for the website; the mobile app is for people who already use the service. */
export const isNativeApp = (): boolean => Capacitor.isNativePlatform();

/**
 * In the Android app the landing page is skipped: a signed-in user goes to their
 * home (/app or /admin), everyone else to login. The website is unchanged.
 */
export const nativeStartGuard: CanActivateFn = () => {
  if (!isNativeApp()) return true;

  const auth = inject(AuthService);
  const router = inject(Router);
  const home = (role: string | undefined) => router.parseUrl(role ? homeFor(role) : '/login');

  const user = auth.currentUser();
  if (user) return home(user.role);
  return auth.getMe().pipe(map((me) => home(me?.role)));
};
