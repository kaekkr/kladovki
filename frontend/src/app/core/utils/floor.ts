/** One wording for floors everywhere: «3 этаж», «Цокольный этаж», «Подземный этаж −1». */
export function floorLabel(floor: number): string {
  if (floor < 0) return `Подземный этаж −${Math.abs(floor)}`;
  if (floor === 0) return 'Цокольный этаж';
  return `${floor} этаж`;
}
