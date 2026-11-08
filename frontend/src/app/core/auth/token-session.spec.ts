import { describe, expect, it } from 'vitest';

import { attachesToken, usesTokenAuth } from './token-session';

describe('token session (mobile app)', () => {
  const api = 'https://kladovki.onrender.com/api/v1';

  it('uses a bearer token only when the app runs on another origin than the API', () => {
    expect(usesTokenAuth(api, 'https://localhost')).toBe(true); // Capacitor Android
    expect(usesTokenAuth(api, 'capacitor://localhost')).toBe(true); // Capacitor iOS
    expect(usesTokenAuth(api, 'https://kladovki.onrender.com')).toBe(false); // website: HttpOnly cookie
  });

  it('never sends the token to other hosts', () => {
    expect(attachesToken(`${api}/rentals/my`, api)).toBe(true);
    expect(attachesToken('https://api.qrserver.com/v1/create-qr-code/', api)).toBe(false);
    expect(attachesToken('https://kladovki.onrender.com.evil.kz/api/v1/x', api)).toBe(false);
  });
});
