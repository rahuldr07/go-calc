<script lang="ts">
  import { defaultKeymap, history, historyKeymap } from "@codemirror/commands"
  import { EditorState } from "@codemirror/state"
  import { EditorView, drawSelection, keymap, placeholder } from "@codemirror/view"
  import { onMount } from "svelte"
  import { acceptCompletion, completionKeymap } from "@codemirror/autocomplete"
  import { completion } from "../editor/complete"
  import { vocabulary } from "../state/vocabulary.svelte"
  import { caretLine, focused, lineField, selectedField, setFocused, setLines, setSelected, valueField } from "../editor/notepad"
  import { dismissRestore, restoreOwn, select, setSource, sheet } from "../state/sheet.svelte"

  let host: HTMLDivElement

  const tally = $derived.by(() => {
    const counted = sheet.lines.filter((l) => l.text !== "" || l.status === "error")
    const refused = counted.filter((l) => l.status === "error").length
    if (counted.length === 0) return ""
    const answers = `${counted.length} ${counted.length === 1 ? "line" : "lines"}`
    return refused === 0 ? answers : `${answers} · ${refused} refused`
  })
  let editor: EditorView | undefined = $state()

  export function insert(text: string) {
    if (!editor) return
    const at = editor.state.selection.main
    editor.dispatch({
      changes: { from: at.from, to: at.to, insert: text },
      selection: { anchor: at.from + text.length },
    })
    editor.focus()
  }

  export function press(command: "back" | "clear" | "newline") {
    if (!editor) return
    const at = editor.state.selection.main
    if (command === "newline") return insert("\n")
    if (command === "back") {
      const from = at.empty ? Math.max(0, at.from - 1) : at.from
      editor.dispatch({ changes: { from, to: at.to }, selection: { anchor: from } })
      return editor.focus()
    }
    const line = editor.state.doc.lineAt(at.head)
    editor.dispatch({ changes: { from: line.from, to: line.to }, selection: { anchor: line.from } })
    editor.focus()
  }

  function inspect(event: Event) {
    const at = (event as CustomEvent<number>).detail
    if (typeof at !== "number") return
    select(sheet.selected === at ? null : at)
    if (!editor || at >= editor.state.doc.lines) return
    const line = editor.state.doc.line(at + 1)
    editor.dispatch({ selection: { anchor: line.to } })
    editor.focus()
    requestAnimationFrame(() =>
      requestAnimationFrame(() => editor?.dispatch({ effects: EditorView.scrollIntoView(line.from, { y: "nearest", yMargin: 24 }) })),
    )
  }

  onMount(() => {
    editor = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: sheet.source,
        extensions: [
          history(),
          drawSelection({ cursorBlinkRate: 1100 }),
          keymap.of([{ key: "Tab", run: acceptCompletion }, ...completionKeymap, ...defaultKeymap, ...historyKeymap]),
          completion(() => vocabulary.words),
          caretLine,
          focused,
          lineField,
          selectedField,
          valueField,
          EditorView.lineWrapping,
          placeholder("12.3 ± 0.2 m · 100 km / 2 h · 18% of 500"),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) void setSource(u.state.doc.toString())
          }),
          EditorView.focusChangeEffect.of((_, on) => setFocused.of(on)),
        ],
      }),
    })
    host.addEventListener("inspectline", inspect)
    return () => {
      host.removeEventListener("inspectline", inspect)
      editor?.destroy()
    }
  })

  $effect(() => {
    const source = sheet.source
    if (!editor || editor.state.doc.toString() === source) return
    editor.dispatch({
      changes: { from: 0, to: editor.state.doc.length, insert: source },
      selection: { anchor: source.length },
      scrollIntoView: true,
    })
  })

  $effect(() => {
    const lines = sheet.lines
    editor?.dispatch({ effects: setLines.of(lines) })
  })

  $effect(() => {
    const at = sheet.selected
    editor?.dispatch({ effects: setSelected.of(at) })
  })
</script>

<div class="sheet-head">
  <h2>Lab notes</h2>
  {#if sheet.restorable}
    <span class="restore">
      opened from a link
      <button type="button" class="chip" onclick={() => void restoreOwn()}>restore my sheet</button>
      <button type="button" class="chip" onclick={dismissRestore} aria-label="Keep this sheet">keep</button>
    </span>
  {/if}
  <span class="tally">{tally}</span>
</div>
<div class="colheads" aria-hidden="true">
  <span></span><span>expression</span><span>result</span><span>trust</span>
</div>
<div class="scroll" bind:this={host}></div>
