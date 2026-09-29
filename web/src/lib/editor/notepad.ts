import { StateEffect, StateField } from "@codemirror/state"
import { Decoration, EditorView, WidgetType, type DecorationSet } from "@codemirror/view"
import type { Line } from "../engine/schema"

export const setLines = StateEffect.define<readonly Line[]>()
export const setFocused = StateEffect.define<boolean>()

const VALUE_WIDTH = 48

function clipped(text: string): { shown: string; full: boolean } {
  if (text.length <= VALUE_WIDTH) return { shown: text, full: true }
  return { shown: `${text.slice(0, VALUE_WIDTH - 1)}…`, full: false }
}

const dots: Record<string, string> = {
  proven: "••••••",
  estimate: "••••○○",
  held: "○○○○○○",
}

function statusClass(status: Line["status"]): string {
  if (status === "proven") return "st proven"
  if (status === "estimate") return "st est"
  if (status === "held") return "st hold"
  if (status === "error") return "st err"
  return "st"
}

class ValueWidget extends WidgetType {
  constructor(
    readonly line: Line,
    readonly editing: boolean,
  ) {
    super()
  }

  eq(other: ValueWidget) {
    return (
      other.line.text === this.line.text &&
      other.line.status === this.line.status &&
      other.line.error === this.line.error &&
      other.editing === this.editing
    )
  }

  toDOM() {
    const wrap = document.createElement("span")
    wrap.className = "val"
    wrap.dataset.line = String(this.line.n)
    wrap.addEventListener("mousedown", (event) => {
      event.preventDefault()
      event.stopPropagation()
      wrap.dispatchEvent(new CustomEvent("inspectline", { bubbles: true, detail: this.line.n }))
    })
    if (this.line.status === "error") {
      if (this.editing) return wrap
      const err = document.createElement("span")
      err.className = "num err-text"
      const message = clipped(this.line.error ?? "refused")
      err.textContent = message.shown
      err.title = this.line.error ?? "refused"
      const st = document.createElement("span")
      st.className = statusClass(this.line.status)
      st.textContent = "refused"
      wrap.append(err, st)
      return wrap
    }
    const num = document.createElement("span")
    num.className = "num"
    const value = clipped(this.line.text)
    num.textContent = value.shown
    if (!value.full) {
      num.classList.add("clipped")
      num.title = `${this.line.text.length} characters — open the line to read it in full`
    }
    const st = document.createElement("span")
    st.className = statusClass(this.line.status)
    st.textContent = dots[this.line.status] ?? this.line.status
    st.title = this.line.status
    wrap.append(num, st)
    wrap.tabIndex = 0
    wrap.setAttribute("role", "button")
    return wrap
  }

  ignoreEvent() {
    return false
  }
}

class NoteWidget extends WidgetType {
  constructor(readonly items: { label: string; text: string }[]) {
    super()
  }

  eq(other: NoteWidget) {
    return JSON.stringify(other.items) === JSON.stringify(this.items)
  }

  toDOM() {
    const holder = document.createElement("div")
    holder.className = "notes"
    for (const item of this.items) {
      const note = document.createElement("div")
      note.className = "note"
      const label = document.createElement("b")
      label.textContent = item.label
      note.append(label, document.createTextNode(item.text))
      holder.append(note)
    }
    return holder
  }
}

function noteItems(line: Line): { label: string; text: string }[] {
  const items: { label: string; text: string }[] = []
  for (const note of line.notes ?? []) {
    items.push({ label: "read as", text: note.text.replace(/^read as /, "") })
  }
  for (const qualifier of line.qualifiers ?? []) {
    items.push({ label: "digits", text: qualifier })
  }
  if (line.warning) items.push({ label: "signal lost", text: line.warning })
  if (line.stopped) items.push({ label: "stopped", text: line.stopped })
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
  doc: { lines: number; line(n: number): { to: number } },
) {
  const marks = []
  for (const line of lines) {
    if (line.n < 0 || line.n >= doc.lines) continue
    const at = doc.line(line.n + 1).to
    const editing = editable && line.n === caret
    marks.push(Decoration.widget({ widget: new ValueWidget(line, editing), side: 1 }).range(at))
    if (editing) continue
    const items = noteItems(line)
    if (items.length === 0) continue
    marks.push(Decoration.widget({ widget: new NoteWidget(items), side: 2, block: true }).range(at))
  }
  return Decoration.set(marks, true)
}

export const valueField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(marks, tr) {
    const linesChanged = tr.effects.some((e) => e.is(setLines) || e.is(setFocused))
    const caretMoved = tr.startState.field(caretLine, false) !== tr.state.field(caretLine, false)
    if (!linesChanged && !caretMoved && !tr.docChanged) return marks
    return build(
      tr.state.field(lineField),
      tr.state.field(caretLine),
      tr.state.field(focused),
      tr.state.doc,
    )
  },
  provide: (field) => EditorView.decorations.from(field),
})

export const notepadTheme = EditorView.theme({
  "&": { color: "var(--ivory)", background: "transparent", fontSize: "14px", height: "100%" },
  ".cm-scroller": { fontFamily: "var(--mono)", lineHeight: "1.55", overflow: "auto" },
  ".cm-content": { padding: "4px 0 40vh", caretColor: "var(--signal)" },
  ".cm-line": { padding: "2px 0 2px 18px", position: "relative" },
  ".cm-line::before": {
    content: '"›"',
    position: "absolute",
    left: "0",
    color: "var(--dust)",
  },
  ".cm-activeLine": { backgroundColor: "transparent" },
  "&.cm-focused": { outline: "none" },
  ".cm-cursor": { borderLeftColor: "var(--signal)", borderLeftWidth: "2px" },
  ".cm-selectionBackground, ::selection": { backgroundColor: "var(--signal)" },
  ".cm-gutters": { display: "none" },
})
