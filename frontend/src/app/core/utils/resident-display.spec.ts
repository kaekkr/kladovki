import { describe, expect, it } from 'vitest';

import {
  canBook,
  propertyLabel,
  residentStatusLabel,
  verificationProblem,
} from './resident-display';

describe('resident display', () => {
  it('labels every status in Russian', () => {
    expect(residentStatusLabel('pending')).toBe('На проверке');
    expect(residentStatusLabel('active')).toBe('Подтверждён');
    expect(residentStatusLabel('rejected')).toBe('Отклонён');
    expect(residentStatusLabel('blocked')).toBe('Заблокирован');
  });

  it('describes the property of a person and of a company', () => {
    expect(propertyLabel({ owner_type: 'person', premises_kind: 'flat', premises_number: '45' })).toBe('Кв. 45');
    expect(
      propertyLabel({
        owner_type: 'company',
        premises_kind: 'commercial',
        premises_number: '3',
        company_name: 'ТОО Аптека',
        company_bin: '180940012345',
      }),
    ).toBe('Помещение 3 · ТОО Аптека (БИН 180940012345)');
  });

  it('only approved residents (and admins) can book', () => {
    expect(canBook({ role: 'resident', status: 'active' })).toBe(true);
    expect(canBook({ role: 'resident', status: 'pending' })).toBe(false);
    expect(canBook({ role: 'resident', status: 'blocked' })).toBe(false);
    expect(canBook({ role: 'admin', status: 'active' })).toBe(true);
    expect(canBook(null)).toBe(false);
  });

  it('validates what the user declares', () => {
    const person = { owner_type: 'person' as const, premises_kind: 'flat' as const, premises_number: '45' };
    expect(verificationProblem(person)).toBeNull();
    expect(verificationProblem({ ...person, premises_number: '  ' })).toMatch(/номер/);
    const company = { ...person, owner_type: 'company' as const, company_bin: '180940012345', company_name: 'ТОО А' };
    expect(verificationProblem(company)).toBeNull();
    expect(verificationProblem({ ...company, company_bin: '1809' })).toMatch(/БИН/);
    expect(verificationProblem({ ...company, company_name: '' })).toMatch(/название/);
  });
});
