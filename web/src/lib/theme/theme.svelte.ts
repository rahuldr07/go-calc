import { DEFAULT_THEME, THEME_KEY, THEMES, cssFor, getTheme, type ThemeId } from "./themes"

function saved(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function keep(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    return
  }
}

function firstTheme(): ThemeId {
  const stored = saved(THEME_KEY)
  if (stored && stored in THEMES) return stored as ThemeId
  try {
    if (matchMedia("(prefers-color-scheme: dark)").matches) return "carbon"
  } catch {
    return DEFAULT_THEME
  }
  return DEFAULT_THEME
}

export const theme = $state({ id: firstTheme() })

export function current() {
  return getTheme(theme.id)
}

export function setTheme(id: ThemeId) {
  theme.id = id
  keep(THEME_KEY, id)
}

export function apply() {
  const active = current()
  const root = document.documentElement
  root.setAttribute("style", cssFor(active))
  root.dataset.theme = active.mode
  root.dataset.world = active.id
}
