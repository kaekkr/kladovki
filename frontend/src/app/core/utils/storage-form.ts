// Same limits as backend/internal/service/limits.go.
export const STORAGE_LIMITS = {
  numberMaxLength: 50,
  areaMin: 0.01,
  areaMax: 10000,
  floorMin: -10,
  floorMax: 200,
  entranceMin: 0,
  entranceMax: 100,
} as const;

export interface StorageFormValue {
  number: string;
  area: number | null;
  floor: number | null;
  entrance: number | null;
}

function intInRange(value: number | null, min: number, max: number): boolean {
  return value !== null && Number.isInteger(value) && value >= min && value <= max;
}

/** Returns a Russian message for the first invalid field, or null. */
export function storageFormError(v: StorageFormValue): string | null {
  const L = STORAGE_LIMITS;
  const number = v.number.trim();
  if (!number) {
    return 'Введите номер кладовой.';
  }
  if ([...number].length > L.numberMaxLength) {
    return `Номер кладовой — не длиннее ${L.numberMaxLength} символов.`;
  }
  const area = v.area === null ? null : Math.round(v.area * 100) / 100;
  if (area === null || !Number.isFinite(area) || area < L.areaMin || area > L.areaMax) {
    return `Площадь: от ${L.areaMin} до ${L.areaMax} м².`;
  }
  if (!intInRange(v.floor, L.floorMin, L.floorMax)) {
    return `Этаж: целое число от ${L.floorMin} до ${L.floorMax}.`;
  }
  if (!intInRange(v.entrance, L.entranceMin, L.entranceMax)) {
    return `Подъезд: целое число от ${L.entranceMin} до ${L.entranceMax}.`;
  }
  return null;
}
