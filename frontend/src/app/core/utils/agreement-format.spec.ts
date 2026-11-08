import { describe, expect, it } from 'vitest';

import { agreementBlocks } from './agreement-format';

describe('agreementBlocks', () => {
  it('splits the agreement into section headings, numbered clauses and plain text', () => {
    const text = [
      '(для Пользователей-Жителей)',
      '1. Общие положения',
      '1.1. Настоящее Соглашение регулирует…',
      '',
      '5.2. Задолженность более 2 (двух) месяцев:',
      'ОСИ имеет право расторгнуть Договор.',
      '10. Заключение',
    ].join('\n');
    expect(agreementBlocks(text)).toEqual([
      { kind: 'text', text: '(для Пользователей-Жителей)' },
      { kind: 'heading', text: '1. Общие положения' },
      { kind: 'clause', text: '1.1. Настоящее Соглашение регулирует…' },
      { kind: 'clause', text: '5.2. Задолженность более 2 (двух) месяцев:' },
      { kind: 'text', text: 'ОСИ имеет право расторгнуть Договор.' },
      { kind: 'heading', text: '10. Заключение' },
    ]);
  });
});
