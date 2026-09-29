<script lang="ts">
  import { Tabs } from "bits-ui"
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
    [...sheet.lines].reverse().find((line) => line.status !== "error" && line.text !== "")?.text ?? "—",
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
    if (text === "ans") return insert(answer === "—" ? "" : answer)
    insert(text)
  }
</script>

<section class="pad" aria-label="Calculator keypad">
  <div class="pad-in">
    <div class="entry"><span class="now"></span><span class="ans">ans <b>{answer}</b></span></div>
    <Tabs.Root bind:value={group}>
      <Tabs.List class="groups">
        {#each GROUP_NAMES as [id, label] (id)}
          <Tabs.Trigger value={id}>{label}</Tabs.Trigger>
        {/each}
      </Tabs.List>
      {#each GROUP_NAMES as [id] (id)}
        <Tabs.Content value={id}>
          <div class="grid">
            {#each PAD_GROUPS[id] as key (key[0])}
              <button
                type="button"
                class="k"
                class:dim={key[2] === "held"}
                aria-disabled={key[2] === "held"}
                onclick={() => tap(key)}
              >{key[0]}</button>
            {/each}
          </div>
        </Tabs.Content>
      {/each}
    </Tabs.Root>
    <div class="core">
      {#each CORE as key (key[0])}
        <button type="button" class="k" class:act={key[2] === "act"} onclick={() => tap(key)}>{key[0]}</button>
      {/each}
    </div>
    <div class="pad-note">{note}</div>
  </div>
</section>
