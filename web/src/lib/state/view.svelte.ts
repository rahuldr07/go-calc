export type Mode = "notepad" | "terminal"

function saved(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function remember(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    return
  }
}

function matches(query: string): boolean {
  try {
    return matchMedia(query).matches
  } catch {
    return false
  }
}

function firstKeypad(): boolean {
  const stored = saved("gocalc-keypad")
  if (stored === "open") return true
  if (stored === "closed") return false
  return matches("(max-width: 860px)") && !matches("(min-width: 861px)")
}

export const view = $state({
  mode: "notepad" as Mode,
  keypad: firstKeypad(),
  layer: false,
  studio: false,
  help: false,
})

export function setMode(mode: Mode) {
  view.mode = mode
  if (mode === "terminal") view.layer = false
}

export function toggleKeypad() {
  view.keypad = !view.keypad
  remember("gocalc-keypad", view.keypad ? "open" : "closed")
}

export function toggleHelp() {
  view.help = !view.help
}

export function openLayer(open: boolean) {
  view.layer = open
}
