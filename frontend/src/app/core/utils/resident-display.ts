import { OwnerType, PremisesKind, UserStatus } from '../auth/auth.models';

export interface PropertyInfo {
  owner_type: OwnerType;
  premises_kind: PremisesKind;
  premises_number: string;
  company_bin?: string;
  company_name?: string;
}

export function residentStatusLabel(status: UserStatus): string {
  switch (status) {
    case 'pending':
      return 'На проверке';
    case 'active':
      return 'Подтверждён';
    case 'rejected':
      return 'Отклонён';
    case 'blocked':
      return 'Заблокирован';
  }
}

/** "Кв. 45" or "Помещение 3 · ТОО Аптека (БИН …)". */
export function propertyLabel(p: PropertyInfo): string {
  const place = (p.premises_kind === 'commercial' ? 'Помещение ' : 'Кв. ') + p.premises_number;
  return p.owner_type === 'company' ? `${place} · ${p.company_name} (БИН ${p.company_bin})` : place;
}

/** Only owners confirmed by the ОСИ may book or renew (user agreement п. 2.1). */
export function canBook(user: { role: string; status?: UserStatus } | null): boolean {
  if (!user) return false;
  return user.role !== 'resident' || (user.status ?? 'active') === 'active';
}

/** Same rules as the server; returns a message or null. */
export function verificationProblem(p: PropertyInfo): string | null {
  if (!p.premises_number.trim()) return 'Укажите номер квартиры или помещения';
  if (p.owner_type === 'company') {
    if (!/^\d{12}$/.test((p.company_bin ?? '').trim())) return 'БИН компании должен состоять из 12 цифр';
    if (!(p.company_name ?? '').trim()) return 'Укажите название компании';
  }
  return null;
}
