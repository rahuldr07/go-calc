<script lang="ts">
  import { sheet } from "../state/sheet.svelte"

  const counted = $derived({
    lines: sheet.lines.filter((l) => l.text !== "" || l.error).length,
    errors: sheet.lines.filter((l) => l.status === "error").length,
    measured: sheet.lines.filter((l) => l.uncertainty).length,
  })
</script>

<footer class="foot">
  <span class:failed={sheet.engineStatus === "failed"}
    >{sheet.engineStatus === "failed" ? sheet.failure : `wasm · codata 2022 · ${sheet.digits} digits`}</span
  >
  <span>{counted.lines} lines · {counted.measured} measured · {counted.errors} refused</span>
</footer>
