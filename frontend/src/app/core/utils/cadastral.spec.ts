import { describe, expect, it } from 'vitest';
import { parsePremisesCadastral, premisesPlaceLabel } from './cadastral';

describe('parsePremisesCadastral', () => {
  it('splits the ЖК number and the unit, normalizing spaces and case', () => {
    expect(parsePremisesCadastral(' 21:319:021:848:8:вп-2 ')).toEqual({
      cadastral: '21:319:021:848:8:ВП-2',
      complex: '21:319:021:848:8',
      unit: 'ВП-2',
      kind: 'commercial',
    });
  });

  it('treats a numeric unit as a flat', () => {
    expect(parsePremisesCadastral('21:319:021:848:8:154')?.kind).toBe('flat');
  });

  it('rejects malformed numbers', () => {
    for (const bad of ['', '21:319', '21:319:021', '21:319:021:848:', 'ab:cd:ef:gh:1', '21:319:021:848:8:ВП 2!']) {
      expect(parsePremisesCadastral(bad)).toBeNull();
    }
  });
});

describe('premisesPlaceLabel', () => {
  it('says what the user owns', () => {
    expect(premisesPlaceLabel({ unit: '154', kind: 'flat' })).toBe('Квартира 154');
    expect(premisesPlaceLabel({ unit: 'ВП-2', kind: 'commercial' })).toBe('Нежилое помещение ВП-2');
  });
});
