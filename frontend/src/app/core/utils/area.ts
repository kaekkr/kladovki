/** «3,5 м²», «12 м²» — Russian decimal comma, at most two decimals. */
export function formatArea(area: number | null | undefined): string {
  if (area == null || !Number.isFinite(area)) return '—';
  const text = (Math.round(area * 100) / 100).toString().replace('.', ',');
  return `${text} м²`;
}
