import { describe, expect, it } from 'vitest';

import { formatArea } from './area';

describe('formatArea', () => {
  it('uses a decimal comma', () => {
    expect(formatArea(3.5)).toBe('3,5 м²');
    expect(formatArea(12)).toBe('12 м²');
    expect(formatArea(2.255)).toBe('2,26 м²');
  });

  it('shows a dash when the area is unknown', () => {
    expect(formatArea(null)).toBe('—');
    expect(formatArea(undefined)).toBe('—');
    expect(formatArea(Number.NaN)).toBe('—');
  });
});
