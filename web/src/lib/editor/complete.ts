import { autocompletion, type Completion, type CompletionContext, type CompletionResult } from "@codemirror/autocomplete"
import type { Extension } from "@codemirror/state"

export interface FunctionDoc {
  signature: string
  summary: string
  category: string
  example: string
}

export interface Vocabulary {
  functions: string[]
  constants: Record<string, string>
  units: string[]
  docs: Record<string, FunctionDoc>
  categories: string[]
}

export const EMPTY_VOCABULARY: Vocabulary = { functions: [], constants: {}, units: [], docs: {}, categories: [] }

function distance(a: string, b: string): number {
  const rows = a.length + 1
  const cols = b.length + 1
  const d: number[][] = Array.from({ length: rows }, () => new Array<number>(cols).fill(0))
  for (let i = 0; i < rows; i++) d[i][0] = i
  for (let j = 0; j < cols; j++) d[0][j] = j
  for (let i = 1; i < rows; i++) {
    for (let j = 1; j < cols; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1
      d[i][j] = Math.min(d[i - 1][j] + 1, d[i][j - 1] + 1, d[i - 1][j - 1] + cost)
      if (i > 1 && j > 1 && a[i - 1] === b[j - 2] && a[i - 2] === b[j - 1]) {
        d[i][j] = Math.min(d[i][j], d[i - 2][j - 2] + 1)
      }
    }
  }
  return d[a.length][b.length]
}

function tolerance(word: string): number {
  if (word.length <= 3) return 1
  if (word.length <= 6) return 2
  return 3
}

function score(word: string, candidate: string): number | null {
  const a = word.toLowerCase()
  const b = candidate.toLowerCase()
  if (b.startsWith(a)) return 0
  if (b.includes(a)) return 1
  const d = distance(a, b)
  return d <= tolerance(a) ? 1 + d : null
}

function entries(vocab: Vocabulary): Completion[] {
  const out: Completion[] = []
  for (const name of vocab.functions) {
    out.push({ label: name, type: "function", apply: `${name}(`, detail: vocab.docs[name]?.signature ?? "function", info: vocab.docs[name]?.summary })
  }
  for (const [symbol, meaning] of Object.entries(vocab.constants)) {
    out.push({ label: symbol, type: "constant", detail: meaning })
  }
  for (const unit of vocab.units) {
    out.push({ label: unit, type: "unit", detail: "unit" })
  }
  return out
}

export function completionsFor(vocab: () => Vocabulary) {
  return (context: CompletionContext): CompletionResult | null => {
    const word = context.matchBefore(/[A-Za-z_][A-Za-z0-9_]*/)
    if (!word || (word.from === word.to && !context.explicit)) return null
    const typed = word.text
    const ranked: { option: Completion; rank: number }[] = []
    for (const option of entries(vocab())) {
      const rank = score(typed, option.label)
      if (rank === null) continue
      ranked.push({ option, rank })
    }
    if (ranked.length === 0) return null
    ranked.sort((a, b) => a.rank - b.rank || a.option.label.length - b.option.label.length)
    return {
      from: word.from,
      options: ranked.slice(0, 24).map((r) => r.option),
      filter: false,
    }
  }
}

export function completion(vocab: () => Vocabulary): Extension {
  return autocompletion({
    override: [completionsFor(vocab)],
    activateOnTyping: true,
    icons: false,
    maxRenderedOptions: 24,
  })
}
