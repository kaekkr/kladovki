export type AgreementDoc = 'resident_terms' | 'osi_rules';
export type AgreementLang = 'ru' | 'kk';

export interface Agreement {
  doc: AgreementDoc;
  version: string;
  lang: AgreementLang;
  title: string;
  text: string;
}

export type AgreementStatus = Record<AgreementDoc, { version: string; accepted: boolean }>;
