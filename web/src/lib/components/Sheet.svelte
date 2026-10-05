<script lang="ts">
  import { defaultKeymap, history, historyKeymap } from "@codemirror/commands"
  import { EditorState } from "@codemirror/state"
  import { EditorView, keymap, placeholder } from "@codemirror/view"
  import { onMount } from "svelte"
  import { caretLine, focused, lineField, selectedField, setFocused, setLines, setSelected, valueField } from "../editor/notepad"
  import { select, setSource, sheet } from "../state/sheet.svelte"

  let host: HTMLDivElement
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
    if (typeof at === "number") select(at)
  }

  onMount(() => {
    editor = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: sheet.source,
        extensions: [
          history(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
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
  <span class="pin">{sheet.selected === null ? "nothing pinned" : `lens pinned to line ${sheet.selected + 1}`}</span>
</div>
<div class="colheads" aria-hidden="true">
  <span></span><span>expression</span><span>result</span><span>trust</span>
</div>
<div class="scroll" bind:this={host}></div>
