export type Mode = "notepad" | "terminal"
export type Theme = "dark" | "light"

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

function firstTheme(): Theme {
  const stored = saved("gocalc-theme")
  if (stored === "light" || stored === "dark") return stored
  return matches("(prefers-color-scheme: light)") ? "light" : "dark"
}

function firstKeypad(): boolean {
  const stored = saved("gocalc-keypad")
  if (stored === "open") return true
  if (stored === "closed") return false
  return matches("(max-width: 860px)")
}

export const view = $state({
  mode: "notepad" as Mode,
  theme: firstTheme(),
  keypad: firstKeypad(),
  layer: false,
})

export function setMode(mode: Mode) {
  view.mode = mode
  if (mode === "terminal") view.layer = false
}

export function toggleTheme() {
  view.theme = view.theme === "light" ? "dark" : "light"
  remember("gocalc-theme", view.theme)
}

export function toggleKeypad() {
  view.keypad = !view.keypad
  remember("gocalc-keypad", view.keypad ? "open" : "closed")
}

export function openLayer(open: boolean) {
  view.layer = open
}
