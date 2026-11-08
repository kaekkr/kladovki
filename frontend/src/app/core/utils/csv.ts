const BOM = '﻿';

function cell(v: string | number): string {
  const s = String(v);
  return /[;"\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

/** CSV that Excel (Russian locale) opens correctly: UTF-8 BOM, ";" separators, CRLF. */
export function toCsv(header: string[], rows: (string | number)[][]): string {
  return BOM + [header, ...rows].map((r) => r.map(cell).join(';')).join('\r\n');
}

/** Starts a browser download of `data` as `filename`. */
export function downloadFile(data: Blob | string, filename: string): void {
  const blob = typeof data === 'string' ? new Blob([data], { type: 'text/csv;charset=utf-8' }) : data;
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

const KZ_OFFSET_MS = 5 * 3600 * 1000;

/** Is an ISO time within [from, to] (YYYY-MM-DD, inclusive, Kazakhstan days)? Empty bounds = open. */
export function inPeriod(iso: string, from: string, to: string): boolean {
  const day = new Date(new Date(iso).getTime() + KZ_OFFSET_MS).toISOString().slice(0, 10);
  return (!from || day >= from) && (!to || day <= to);
}
