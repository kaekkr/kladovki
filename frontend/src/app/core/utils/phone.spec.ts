import { describe, expect, it } from 'vitest';

import { formatPhone, normalizePhone, PHONE_PROBLEM, phoneProblem } from './phone';

describe('normalizePhone', () => {
  it.each([
    ['+7 777 123 45 67', '+77771234567'],
    ['+7 (777) 123-45-67', '+77771234567'],
    ['8 777 123 45 67', '+77771234567'],
    ['87771234567', '+77771234567'],
    ['77771234567', '+77771234567'],
    ['7771234567', '+77771234567'],
    ['  +7-701-000-00-00 ', '+77010000000'],
  ])('%s → %s', (raw, canonical) => {
    expect(normalizePhone(raw)).toBe(canonical);
  });

  it.each([
    '',
    '+7 327 123 45 67', // not a mobile code (7xx)
    '+7 777 123 45', // too short
    '+7 777 123 45 678', // too long
    '+1 777 123 45 67',
    'телефон',
  ])('rejects %s', (raw) => {
    expect(normalizePhone(raw)).toBeNull();
  });
});

describe('phoneProblem', () => {
  it('is null for a valid number and a message otherwise', () => {
    expect(phoneProblem('8 (705) 111-22-33')).toBeNull();
    expect(phoneProblem('123')).toBe(PHONE_PROBLEM);
    expect(phoneProblem('   ')).toBe('Укажите номер телефона');
  });
});

describe('formatPhone', () => {
  it('formats a valid number for display', () => {
    expect(formatPhone('87771234567')).toBe('+7 777 123 45 67');
  });

  it('returns the input when it cannot be parsed', () => {
    expect(formatPhone('abc')).toBe('abc');
    expect(formatPhone('')).toBe('');
  });
});
