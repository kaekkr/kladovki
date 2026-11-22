import { PremisesKind } from '../auth/auth.models';

/**
 * Premises cadastral number, e.g. 21:319:021:848:8:ВП-2.
 * The ЖК is 21:319:021:848:8, the last facet is the flat or non-residential unit.
 * Same rules as backend/internal/cadastral.
 */
export interface PremisesCadastral {
  cadastral: string;
  complex: string;
  unit: string;
  kind: PremisesKind;
}

const MIN_COMPLEX_FACETS = 4;
const DIGITS = /^\d+$/;
const UNIT = /^[\p{L}\d]+(?:[-/][\p{L}\d]+)*$/u;

export function normalizeCadastral(raw: string): string {
  return raw.replace(/\s+/g, '').replace(/:+$/, '').toUpperCase();
}

function validComplex(s: string): boolean {
  const facets = s.split(':');
  return facets.length >= MIN_COMPLEX_FACETS && facets.every((f) => DIGITS.test(f));
}

export function parsePremisesCadastral(raw: string): PremisesCadastral | null {
  const cadastral = normalizeCadastral(raw);
  const i = cadastral.lastIndexOf(':');
  if (i <= 0 || i === cadastral.length - 1) return null;
  const complex = cadastral.slice(0, i);
  const unit = cadastral.slice(i + 1);
  if (!validComplex(complex) || !UNIT.test(unit)) return null;
  return { cadastral, complex, unit, kind: DIGITS.test(unit) ? 'flat' : 'commercial' };
}

export function premisesPlaceLabel(p: { unit: string; kind: PremisesKind }): string {
  return (p.kind === 'flat' ? 'Квартира ' : 'Нежилое помещение ') + p.unit;
}

export const CADASTRAL_EXAMPLE = '21:319:021:848:8:154';
export const CADASTRAL_HINT =
  'Указан в техпаспорте или документе о праве собственности. Последняя часть — номер квартиры или помещения.';

export const CADASTRAL_PROBLEM = `Укажите кадастровый номер помещения полностью, например ${CADASTRAL_EXAMPLE}`;

/** Fields sent to the server for the owned premises (number and kind come from the cadastral number). */
export function premisesFields(p: PremisesCadastral) {
  return { premises_cadastral: p.cadastral, premises_number: p.unit, premises_kind: p.kind };
}
