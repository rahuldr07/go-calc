import { EMPTY_VOCABULARY, type Vocabulary } from "../editor/complete"

export const vocabulary = $state<{ words: Vocabulary }>({ words: EMPTY_VOCABULARY })

export function adopt(raw: string) {
  if (raw === "") return
  try {
    const read = JSON.parse(raw) as Partial<Vocabulary>
    vocabulary.words = {
      functions: read.functions ?? [],
      constants: read.constants ?? {},
      units: read.units ?? [],
      docs: read.docs ?? {},
      categories: read.categories ?? [],
    }
  } catch {
    return
  }
}
