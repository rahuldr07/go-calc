<script lang="ts">
  import { defaultKeymap, history, historyKeymap } from "@codemirror/commands"
  import { EditorState } from "@codemirror/state"
  import { EditorView, keymap, placeholder } from "@codemirror/view"
  import { onMount } from "svelte"
  import { caretLine, lineField, notepadTheme, setLines, valueField } from "../editor/notepad"
  import { select, setSource, sheet } from "../state/sheet.svelte"
  import { openLayer } from "../state/view.svelte"

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

  onMount(() => {
    editor = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: sheet.source,
        extensions: [
          history(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          caretLine,
          lineField,
          valueField,
          notepadTheme,
          EditorView.lineWrapping,
          placeholder("type an expression — 12.3 ± 0.2 m, 100 km / 2 h, 18% of 500"),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) void setSource(update.state.doc.toString())
          }),
        ],
      }),
    })
    return () => editor?.destroy()
  })

  $effect(() => {
    const lines = sheet.lines
    editor?.dispatch({ effects: setLines.of(lines) })
  })

  function inspect(event: MouseEvent | KeyboardEvent) {
    const target = (event.target as HTMLElement | null)?.closest(".val") as HTMLElement | null
    if (!target?.dataset.line) return
    if (event instanceof KeyboardEvent && event.key !== "Enter" && event.key !== " ") return
    event.preventDefault()
    select(Number(target.dataset.line))
    openLayer(true)
  }
</script>

<main class="stage">
  <div class="sheet" bind:this={host} onclick={inspect} onkeydown={inspect} role="presentation"></div>
</main>
