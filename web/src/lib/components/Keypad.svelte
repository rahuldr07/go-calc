<script lang="ts">
  import { CORE, GROUP_NAMES, HELD_NOTE, PAD_GROUPS, type Key } from "../editor/keys"

  interface Props {
    insert: (text: string) => void
    press: (command: "back" | "clear" | "newline") => void
  }
  const { insert, press }: Props = $props()

  let group = $state("sci")
  let note = $state("")

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
    insert(text)
  }
</script>

<section class="pad" aria-label="Keypad">
  <div class="pad-top">
    <div class="pills pad-groups" role="tablist" aria-label="Key groups">
      {#each GROUP_NAMES as [id, label] (id)}
        <button type="button" role="tab" aria-selected={group === id} onclick={() => { group = id; note = "" }}>{label}</button>
      {/each}
    </div>
    <div class="pad-note" role="status">{note}</div>
  </div>
  <div class="pad-keys">
    {#each PAD_GROUPS[group] as key, k (group + k)}
      <button type="button" class="key" class:off={key[2] === "held"} aria-disabled={key[2] === "held"}
        title={key[2] === "held" ? HELD_NOTE : key[1].trim()} onclick={() => tap(key)}>{key[0]}</button>
    {/each}
  </div>
  <div class="pad-core">
    {#each CORE as key (key[0])}
      <button type="button" class="key" class:go={key[2] === "act"} onclick={() => tap(key)}>{key[0]}</button>
    {/each}
  </div>
</section>
