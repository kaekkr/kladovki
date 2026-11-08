export type Role = 'admin' | 'resident';

export function homeFor(role: string): string {
  return role === 'admin' ? '/admin/dashboard' : '/app';
}

/**
 * Where to go after login: only an in-app path of the user's own area
 * (/app… for residents, /admin… for ОСИ). Anything else → null (use home).
 */
export function safeReturnUrl(url: string | null | undefined, role: string): string | null {
  if (!url || !url.startsWith('/') || url.startsWith('//')) return null;
  const area = role === 'admin' ? '/admin' : '/app';
  return url === area || url.startsWith(area + '/') || url.startsWith(area + '?') ? url : null;
}
