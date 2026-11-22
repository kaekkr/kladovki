import { describe, expect, it } from 'vitest';
import { matchesSearch } from './admin-search';

describe('matchesSearch', () => {
  const fields = ['Ержан Активный', '+77001112233', 'Т-12', '990101300011'];

  it('matches any field case-insensitively', () => {
    expect(matchesSearch(fields, 'ерж')).toBe(true);
    expect(matchesSearch(fields, 'т-12')).toBe(true);
    expect(matchesSearch(fields, 'нет такого')).toBe(false);
  });

  it('matches phones however they are typed (8-700…, +7 700…, 700 111)', () => {
    expect(matchesSearch(fields, '8-700-111-22-33')).toBe(true);
    expect(matchesSearch(fields, '+7 700 111')).toBe(true);
    expect(matchesSearch(fields, '1112233')).toBe(true);
  });

  it('empty query matches everything', () => {
    expect(matchesSearch(fields, '  ')).toBe(true);
  });
});
