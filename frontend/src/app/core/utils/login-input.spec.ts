import { describe, expect, it } from 'vitest';

import { classifyLogin, passwordsProblem } from './login-input';

describe('classifyLogin', () => {
  it('recognises IIN, email and phone', () => {
    expect(classifyLogin(' 900101300001 ')).toBe('iin');
    expect(classifyLogin('Admin@Example.com')).toBe('email');
    expect(classifyLogin('+7 700 111 22 33')).toBe('phone');
    expect(classifyLogin('87001112233')).toBe('phone');
    expect(classifyLogin('12345')).toBe('invalid');
    expect(classifyLogin('')).toBe('invalid');
  });
});

describe('passwordsProblem', () => {
  it('checks length and match for first login', () => {
    expect(passwordsProblem('123', '123')).toBe('Пароль должен быть не короче 6 символов');
    expect(passwordsProblem('secret1', 'secret2')).toBe('Пароли не совпадают');
    expect(passwordsProblem('secret1', 'secret1')).toBeNull();
  });
});
