import { registerLocaleData } from '@angular/common';
import localeRu from '@angular/common/locales/ru';
import { beforeAll, describe, expect, it } from 'vitest';

import { formatRuDate, formatRuDayMonth, formatTenge } from './date-format';

beforeAll(() => registerLocaleData(localeRu));

describe('formatRuDate', () => {
  it('uses «d MMMM y» in Kazakhstan time', () => {
    expect(formatRuDate('2027-03-24T10:00:00Z')).toBe('24 марта 2027');
    // 21:00 UTC is already the next day in Almaty.
    expect(formatRuDate('2027-03-23T21:00:00Z')).toBe('24 марта 2027');
  });

  it('shows a dash for missing or broken dates', () => {
    expect(formatRuDate(null)).toBe('—');
    expect(formatRuDate('not a date')).toBe('—');
  });

  it('has a short day-month form', () => {
    expect(formatRuDayMonth('2027-03-24T10:00:00Z')).toBe('24 марта');
  });
});

describe('formatTenge', () => {
  it('groups thousands', () => {
    expect(formatTenge(7500).replace(/\s/g, ' ')).toBe('7 500 ₸');
  });
});
