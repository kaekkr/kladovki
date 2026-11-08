import { describe, expect, it } from 'vitest';

import { homeFor, safeReturnUrl } from './return-url';

describe('safeReturnUrl', () => {
  it('keeps in-app links of the right area (e.g. from a Telegram notification)', () => {
    expect(safeReturnUrl('/app/rentals/r1', 'resident')).toBe('/app/rentals/r1');
    expect(safeReturnUrl('/admin/requests', 'admin')).toBe('/admin/requests');
  });

  it('rejects other areas, external and protocol-relative URLs', () => {
    expect(safeReturnUrl('/admin/requests', 'resident')).toBeNull();
    expect(safeReturnUrl('/app', 'admin')).toBeNull();
    expect(safeReturnUrl('https://evil.example/app', 'resident')).toBeNull();
    expect(safeReturnUrl('//evil.example/app', 'resident')).toBeNull();
    expect(safeReturnUrl('/application', 'resident')).toBeNull();
    expect(safeReturnUrl(null, 'resident')).toBeNull();
  });

  it('has a home per role', () => {
    expect(homeFor('admin')).toBe('/admin/dashboard');
    expect(homeFor('resident')).toBe('/app');
  });
});
