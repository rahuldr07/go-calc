<script lang="ts">
  import { CORE, GROUP_NAMES, HELD_NOTE, PAD_GROUPS, type Key } from "../editor/keys"
  import { sheet } from "../state/sheet.svelte"

  interface Props {
    insert: (text: string) => void
    press: (command: "back" | "clear" | "newline") => void
  }
  const { insert, press }: Props = $props()

  let group = $state("sci")
  let note = $state("")

  const answer = $derived(
    [...sheet.lines].reverse().find((l) => l.status !== "error" && l.text !== "")?.text ?? "",
  )

  function tap(key: Key) {
    const [label, text, state] = key
    if (state === "held") {
      note = `${label} — ${HELD_NOTE}`
      return
    }
    note = ""
    if (text === "BACK") return press("back")
    if (text === "CLEAR") return press("clear")
    if (text === "EQ") return press("newline")
    if (text === "ans") return insert(answer)
    insert(text)
  }
</script>

<section class="pad" aria-label="Keypad">
  <div class="pad-groups" role="tablist">
    {#each GROUP_NAMES as [id, label] (id)}
      <button type="button" role="tab" aria-selected={group === id} data-state={group === id ? "active" : ""}
        onclick={() => (group = id)}>{label}</button>
    {/each}
  </div>
  <div class="pad-keys">
    {#each PAD_GROUPS[group] as key (key[0])}
      <button type="button" class="key" class:off={key[2] === "held"} aria-disabled={key[2] === "held"}
        onclick={() => tap(key)}>{key[0]}</button>
    {/each}
  </div>
  <div class="pad-core">
    {#each CORE as key (key[0])}
      <button type="button" class="key" class:go={key[2] === "act"} onclick={() => tap(key)}>{key[0]}</button>
    {/each}
  </div>
  <div class="pad-note" role="status">{note}</div>
</section>
