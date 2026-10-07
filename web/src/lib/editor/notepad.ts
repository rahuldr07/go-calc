import { StateEffect, StateField } from "@codemirror/state"
import { Decoration, EditorView, WidgetType, type DecorationSet } from "@codemirror/view"
import type { Line, Status } from "../engine/schema"

export const setLines = StateEffect.define<readonly Line[]>()
export const setFocused = StateEffect.define<boolean>()
export const setSelected = StateEffect.define<number | null>()

const VALUE_CHARS = 42

const WORD: Record<Status, string> = {
  exact: "exact",
  proven: "proven",
  estimate: "estimate",
  held: "held",
  error: "error",
}

function clip(text: string): { shown: string; full: boolean } {
  if (text.length <= VALUE_CHARS) return { shown: text, full: true }
  return { shown: `${text.slice(0, VALUE_CHARS - 1)}…`, full: false }
}

class AnswerWidget extends WidgetType {
  constructor(
    readonly line: Line,
    readonly editing: boolean,
  ) {
    super()
  }

  eq(other: AnswerWidget) {
    return (
      other.line.text === this.line.text &&
      other.line.status === this.line.status &&
      other.line.error === this.line.error &&
      other.editing === this.editing
    )
  }

  toDOM() {
    const frag = document.createElement("span")
    frag.className = "answer-group"
    const refused = this.line.status === "error"

    const spine = document.createElement("span")
    spine.className = "spine"
    const rune = document.createElement("span")
    rune.className = `rune ${this.line.status}`
    if (this.line.status === "estimate") rune.textContent = "±"
    if (refused) {
      const mark = document.createElement("i")
      mark.textContent = "×"
      rune.append(mark)
    }
    rune.setAttribute("aria-hidden", "true")
    spine.append(rune)

    const result = document.createElement("span")
    result.className = refused ? "result refused" : "result"
    const value = document.createElement("span")
    value.className = "value"
    const body = refused ? (this.line.error ?? "refused") : this.line.text
    const shown = clip(body)
    value.textContent = refused && this.editing ? "" : shown.shown
    if (!shown.full) {
      result.classList.add("clipped")
      result.title = `${body.length} characters — open the line to read it in full`
    }
    result.append(value)

    const status = document.createElement("span")
    status.className = refused ? "status error" : "status"
    status.textContent = refused && this.editing ? "" : WORD[this.line.status]

    frag.append(spine, result, status)
    frag.dataset.line = String(this.line.n)
    frag.setAttribute("role", "button")
    frag.tabIndex = 0
    frag.addEventListener("mousedown", (event) => {
      event.preventDefault()
      event.stopPropagation()
      frag.dispatchEvent(new CustomEvent("inspectline", { bubbles: true, detail: this.line.n }))
    })
    return frag
  }

  ignoreEvent() {
    return false
  }
}

class AnnotationWidget extends WidgetType {
  constructor(readonly items: { kind: string; label: string; text: string }[]) {
    super()
  }

  eq(other: AnnotationWidget) {
    return JSON.stringify(other.items) === JSON.stringify(this.items)
  }

  toDOM() {
    const holder = document.createElement("div")
    for (const item of this.items) {
      const row = document.createElement("div")
      row.className = item.kind === "warn" ? "note-block" : "note-block read"
      const label = document.createElement("b")
      label.textContent = item.label
      const text = document.createElement("span")
      text.textContent = item.text
      row.append(label, text)
      holder.append(row)
    }
    return holder
  }
}

function annotations(line: Line) {
  const items: { kind: string; label: string; text: string }[] = []
  for (const note of line.notes ?? []) {
    items.push({ kind: "note", label: "read as", text: note.text.replace(/^read as /, "") })
  }
  for (const qualifier of line.qualifiers ?? []) {
    items.push({ kind: "qual", label: "digits", text: qualifier })
  }
  if (line.warning) items.push({ kind: "warn", label: "float64", text: line.warning })
  if (line.stopped) items.push({ kind: "qual", label: "stopped", text: line.stopped })
  return items
}

export const focused = StateField.define<boolean>({
  create: () => false,
  update(current, tr) {
    for (const effect of tr.effects) if (effect.is(setFocused)) return effect.value
    return current
  },
})

export const caretLine = StateField.define<number>({
  create: () => 0,
  update(current, tr) {
    if (!tr.selection && !tr.docChanged) return current
    return tr.state.doc.lineAt(tr.state.selection.main.head).number - 1
  },
})

export const selectedField = StateField.define<number | null>({
  create: () => null,
  update(current, tr) {
    for (const effect of tr.effects) if (effect.is(setSelected)) return effect.value
    return current
  },
})

export const lineField = StateField.define<readonly Line[]>({
  create: () => [],
  update(current, tr) {
    for (const effect of tr.effects) if (effect.is(setLines)) return effect.value
    return current
  },
})

function build(
  lines: readonly Line[],
  caret: number,
  editable: boolean,
  selected: number | null,
  doc: { lines: number; line(n: number): { from: number; to: number } },
) {
  const marks = []
  for (let n = 0; n < doc.lines; n++) {
    const at = doc.line(n + 1)
    const attributes: Record<string, string> = { "data-n": String(n + 1) }
    if (n === selected) attributes.class = "sel"
    marks.push(Decoration.line({ attributes }).range(at.from))
  }
  for (const line of lines) {
    if (line.n < 0 || line.n >= doc.lines) continue
    const at = doc.line(line.n + 1)
    if (at.from === at.to) continue
    const editing = editable && line.n === caret
    marks.push(Decoration.widget({ widget: new AnswerWidget(line, editing), side: 1 }).range(at.to))
    if (editing) continue
    const items = annotations(line)
    if (items.length === 0) continue
    marks.push(Decoration.widget({ widget: new AnnotationWidget(items), side: 2, block: true }).range(at.to))
  }
  return Decoration.set(marks, true)
}

export const valueField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(marks, tr) {
    const changed = tr.effects.some((e) => e.is(setLines) || e.is(setFocused) || e.is(setSelected))
    const moved = tr.startState.field(caretLine, false) !== tr.state.field(caretLine, false)
    if (!changed && !moved && !tr.docChanged) return marks
    return build(
      tr.state.field(lineField),
      tr.state.field(caretLine),
      tr.state.field(focused),
      tr.state.field(selectedField),
      tr.state.doc,
    )
  },
  provide: (field) => EditorView.decorations.from(field),
})
