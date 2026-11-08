import { describe, expect, it } from 'vitest';

import { inPeriod, toCsv } from './csv';

describe('toCsv', () => {
  it('builds an Excel-friendly CSV: BOM, ";" separators, CRLF, quoted when needed', () => {
    const csv = toCsv(['Дата', 'Кладовка', 'Сумма'], [
      ['01.11.2026', '№12', 5000],
      ['02.11.2026', 'ТОО "Аптека"; склад', -1500],
    ]);
    expect(csv.startsWith('﻿')).toBe(true);
    expect(csv.slice(1).split('\r\n')).toEqual([
      'Дата;Кладовка;Сумма',
      '01.11.2026;№12;5000',
      '02.11.2026;"ТОО ""Аптека""; склад";-1500',
    ]);
  });
});

describe('inPeriod', () => {
  it('filters by inclusive Kazakhstan days', () => {
    // 2026-11-01 00:30 in Almaty is still 31 Oct in UTC
    expect(inPeriod('2026-10-31T19:30:00Z', '2026-11-01', '2026-11-30')).toBe(true);
    expect(inPeriod('2026-10-31T18:30:00Z', '2026-11-01', '2026-11-30')).toBe(false);
    expect(inPeriod('2026-11-30T18:59:00Z', '2026-11-01', '2026-11-30')).toBe(true);
    expect(inPeriod('2026-11-30T19:00:00Z', '2026-11-01', '2026-11-30')).toBe(false);
    expect(inPeriod('2026-05-05T10:00:00Z', '', '')).toBe(true);
  });
});
