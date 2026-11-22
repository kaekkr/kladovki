import { describe, expect, it } from 'vitest';

import {
  apiErrorText,
  normalizeCadastral,
  validateAccountForm,
  validateJKForm,
} from './platform-form';

describe('normalizeCadastral', () => {
  it('removes spaces and trailing colons like the server', () => {
    expect(normalizeCadastral(' 21 : 319:021:848:8: ')).toBe('21:319:021:848:8');
    expect(normalizeCadastral('')).toBe('');
  });
});

describe('validateJKForm', () => {
  const ok = { name: 'ЖК Тест', address: 'Астана', cadastral_number: '21:319:021:848:8' };

  it('accepts a complete form, with or without a cadastral number', () => {
    expect(validateJKForm(ok)).toBeNull();
    expect(validateJKForm({ ...ok, cadastral_number: '' })).toBeNull();
  });

  it('asks for the name and address', () => {
    expect(validateJKForm({ ...ok, name: '  ' })).toBe('Укажите название ЖК.');
    expect(validateJKForm({ ...ok, address: '' })).toBe('Укажите адрес ЖК.');
  });

  it('rejects a cadastral number that is not a ЖК number', () => {
    expect(validateJKForm({ ...ok, cadastral_number: '21:319' })).toContain('21:319:021:848:8');
    expect(validateJKForm({ ...ok, cadastral_number: '21:319:021:848:8:ВП-2' })).toContain(
      '21:319:021:848:8',
    );
  });
});

describe('validateAccountForm', () => {
  const ok = { full_name: 'Иванов Иван', email: 'osi@example.kz', phone: '+77011234567' };

  it('accepts a complete account', () => {
    expect(validateAccountForm(ok)).toBeNull();
  });

  it('names the missing or wrong field', () => {
    expect(validateAccountForm({ ...ok, full_name: ' ' })).toBe('Укажите ФИО администратора ОСИ.');
    expect(validateAccountForm({ ...ok, email: 'not-an-email' })).toBe('Укажите корректный email.');
    expect(validateAccountForm({ ...ok, phone: '' })).toBe('Укажите номер телефона.');
  });
});

describe('apiErrorText', () => {
  it('shows the server message when there is one', () => {
    expect(
      apiErrorText({ status: 409, error: { error: 'Этот email уже используется.' } }, 'x'),
    ).toBe('Этот email уже используется.');
  });

  it('explains a lost connection and falls back otherwise', () => {
    expect(apiErrorText({ status: 0, error: null }, 'Не удалось')).toContain('интернет');
    expect(apiErrorText({ status: 500, error: 'boom' }, 'Не удалось')).toBe('Не удалось');
    expect(apiErrorText(undefined, 'Не удалось')).toBe('Не удалось');
  });
});
