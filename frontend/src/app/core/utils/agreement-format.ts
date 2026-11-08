export interface AgreementBlock {
  kind: 'heading' | 'clause' | 'text';
  text: string;
}

/** "1. Общие положения" → heading, "1.1. …" → clause, anything else → text. Blank lines dropped. */
export function agreementBlocks(text: string): AgreementBlock[] {
  return text
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
    .map((line) => ({
      kind: /^\d+\.\d+/.test(line) ? 'clause' : /^\d+\.\s/.test(line) ? 'heading' : 'text',
      text: line,
    }));
}
