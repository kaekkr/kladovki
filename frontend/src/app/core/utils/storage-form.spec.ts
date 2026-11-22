import { describe, expect, it } from 'vitest';

import { storageFormError } from './storage-form';

describe('storageFormError', () => {
  const ok = { number: 'Т-1', area: 2.5, floor: -1, entrance: 1 };

  it('accepts a normal storage room', () => {
    expect(storageFormError(ok)).toBeNull();
    expect(storageFormError({ ...ok, area: 0.01, floor: -10, entrance: 0 })).toBeNull();
    expect(storageFormError({ ...ok, area: 10000, floor: 200, entrance: 100 })).toBeNull();
  });

  it('uses the same bounds as the server', () => {
    expect(storageFormError({ ...ok, number: '  ' })).toContain('номер');
    expect(storageFormError({ ...ok, number: 'Я'.repeat(51) })).toContain('50');
    expect(storageFormError({ ...ok, area: 0.004 })).toContain('Площадь');
    expect(storageFormError({ ...ok, area: 10000.01 })).toContain('Площадь');
    expect(storageFormError({ ...ok, area: null })).toContain('Площадь');
    expect(storageFormError({ ...ok, floor: -11 })).toContain('Этаж');
    expect(storageFormError({ ...ok, floor: 201 })).toContain('Этаж');
    expect(storageFormError({ ...ok, floor: 1.5 })).toContain('Этаж');
    expect(storageFormError({ ...ok, entrance: 101 })).toContain('Подъезд');
    expect(storageFormError({ ...ok, entrance: null })).toContain('Подъезд');
  });
});
