import { describe, expect, it } from 'vitest';

import { floorLabel } from './floor';

describe('floorLabel', () => {
  it('labels every floor the same way (no «Подземный этаж» next to «-2 этаж»)', () => {
    expect(floorLabel(3)).toBe('3 этаж');
    expect(floorLabel(1)).toBe('1 этаж');
    expect(floorLabel(0)).toBe('Цокольный этаж');
    expect(floorLabel(-1)).toBe('Подземный этаж −1');
    expect(floorLabel(-2)).toBe('Подземный этаж −2');
  });
});
