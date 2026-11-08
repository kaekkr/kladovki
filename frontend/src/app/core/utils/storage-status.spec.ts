import { describe, expect, it } from 'vitest';

import { storageStatusLabel } from './storage-status';

describe('storageStatusLabel', () => {
  it('uses the words of the user agreement (п. 3.1)', () => {
    expect(storageStatusLabel('free')).toBe('Свободно');
    expect(storageStatusLabel('locked')).toBe('Забронировано');
    expect(storageStatusLabel('occupied')).toBe('Занято');
  });
});
