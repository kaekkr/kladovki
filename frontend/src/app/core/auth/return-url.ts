export type Role = 'admin' | 'resident' | 'superadmin';

/** The area (URL prefix) each role works in. */
function areaFor(role: string): string {
  if (role === 'admin') return '/admin';
  if (role === 'superadmin') return '/platform';
  return '/app';
}

export function homeFor(role: string): string {
  return role === 'admin' ? '/admin/dashboard' : areaFor(role);
}

/**
 * Where to go after login: only an in-app path of the user's own area
 * (/app… for residents, /admin… for ОСИ, /platform… for the platform
 * operator). Anything else → null (use home).
 */
export function safeReturnUrl(url: string | null | undefined, role: string): string | null {
  if (!url || !url.startsWith('/') || url.startsWith('//')) return null;
  const area = areaFor(role);
  return url === area || url.startsWith(area + '/') || url.startsWith(area + '?') ? url : null;
}
