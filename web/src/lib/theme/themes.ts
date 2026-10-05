export type ThemeId = "mineral" | "carbon" | "blueprint" | "contrast"
export type Mode = "dark" | "light"

export interface Roles {
  canvas: string
  shell: string
  paper: string
  paperDeep: string
  paperSoft: string
  ink: string
  muted: string
  faint: string
  seam: string
  seamStrong: string
  accent: string
  accentSoft: string
  series1: string
  series2: string
  series2Soft: string
  error: string
  warning: string
  track: string
}

export interface Shape {
  outer: number
  inner: number
  control: number
  row: number
  shadow: string
  glint: string
  glowOpacity: number
}

export interface Theme {
  id: ThemeId
  label: string
  note: string
  mode: Mode
  colors: Roles
  shape: Shape
}

export const DEFAULT_THEME: ThemeId = "mineral"
export const THEME_KEY = "gocalc-theme"

export const THEMES: Record<ThemeId, Theme> = {
  mineral: {
    id: "mineral",
    label: "Mineral",
    note: "warm field notebook",
    mode: "light",
    colors: {
      canvas: "#dde2df",
      shell: "#c9d0cb",
      paper: "#f8f9f5",
      paperDeep: "#eef1ec",
      paperSoft: "#f2f4ef",
      ink: "#202521",
      muted: "#59645e",
      faint: "#66706a",
      seam: "rgba(48,64,54,.12)",
      seamStrong: "rgba(48,64,54,.23)",
      accent: "#b65324",
      accentSoft: "#fbf1eb",
      series1: "#a9471f",
      series2: "#2f6664",
      series2Soft: "#d7e6e2",
      error: "#a43d34",
      warning: "#99601f",
      track: "#d9ded9",
    },
    shape: {
      outer: 24,
      inner: 19,
      control: 8,
      row: 50,
      shadow: "0 28px 84px rgba(37,52,43,.19), inset 0 1px rgba(255,255,255,.22)",
      glint: "radial-gradient(circle at 8% 0%, rgba(255,255,255,.72), transparent 26%)",
      glowOpacity: 0.34,
    },
  },
  carbon: {
    id: "carbon",
    label: "Spectrometer",
    note: "dark instrument bay",
    mode: "dark",
    colors: {
      canvas: "#171c19",
      shell: "#2a312d",
      paper: "#202622",
      paperDeep: "#181d1a",
      paperSoft: "#252c27",
      ink: "#edf1ed",
      muted: "#abb4ae",
      faint: "#89958d",
      seam: "rgba(226,238,230,.09)",
      seamStrong: "rgba(226,238,230,.18)",
      accent: "#e17b43",
      accentSoft: "#4a3024",
      series1: "#f08a4f",
      series2: "#67b8b7",
      series2Soft: "#263f3c",
      error: "#ed887f",
      warning: "#e4ad68",
      track: "#353d38",
    },
    shape: {
      outer: 21,
      inner: 16,
      control: 7,
      row: 49,
      shadow: "0 30px 90px rgba(0,0,0,.26), inset 0 1px rgba(255,255,255,.07)",
      glint: "radial-gradient(circle at 84% -8%, rgba(225,123,67,.26), transparent 34%)",
      glowOpacity: 0.52,
    },
  },
  blueprint: {
    id: "blueprint",
    label: "Blueprint",
    note: "technical drafting table",
    mode: "light",
    colors: {
      canvas: "#102e63",
      shell: "#2b4d82",
      paper: "#eef4ff",
      paperDeep: "#dfe9fb",
      paperSoft: "#f6f8ff",
      ink: "#12213f",
      muted: "#4a5e7f",
      faint: "#5e6f90",
      seam: "rgba(26,59,116,.12)",
      seamStrong: "rgba(26,59,116,.24)",
      accent: "#2454d6",
      accentSoft: "#ceddff",
      series1: "#1f50cc",
      series2: "#a53f35",
      series2Soft: "#f1dcd8",
      error: "#a53343",
      warning: "#76551c",
      track: "#cbd8ed",
    },
    shape: {
      outer: 16,
      inner: 12,
      control: 6,
      row: 48,
      shadow: "0 32px 86px rgba(0,17,53,.25), inset 0 1px rgba(255,255,255,.12)",
      glint:
        "linear-gradient(rgba(255,255,255,.07) 1px, transparent 1px) 0 0/28px 28px, linear-gradient(90deg, rgba(255,255,255,.07) 1px, transparent 1px) 0 0/28px 28px",
      glowOpacity: 0,
    },
  },
  contrast: {
    id: "contrast",
    label: "Proof",
    note: "print-safe analytical",
    mode: "light",
    colors: {
      canvas: "#111311",
      shell: "#d9ddd9",
      paper: "#fbfcf8",
      paperDeep: "#f0f2ee",
      paperSoft: "#ffffff",
      ink: "#111411",
      muted: "#3d463f",
      faint: "#5a645d",
      seam: "rgba(17,20,17,.18)",
      seamStrong: "rgba(17,20,17,.42)",
      accent: "#064fc4",
      accentSoft: "#dce8ff",
      series1: "#064fc4",
      series2: "#8a245d",
      series2Soft: "#f3dfeb",
      error: "#9f1717",
      warning: "#7b4b00",
      track: "#c9ceca",
    },
    shape: {
      outer: 10,
      inner: 7,
      control: 4,
      row: 54,
      shadow: "none",
      glint: "none",
      glowOpacity: 0,
    },
  },
}

export const THEME_LIST: Theme[] = Object.values(THEMES)

export function getTheme(id: string | null | undefined): Theme {
  return THEMES[(id ?? "") as ThemeId] ?? THEMES[DEFAULT_THEME]
}

export function cssFor(theme: Theme): string {
  const c = theme.colors
  const s = theme.shape
  return [
    `--canvas:${c.canvas}`,
    `--shell:${c.shell}`,
    `--paper:${c.paper}`,
    `--paper-deep:${c.paperDeep}`,
    `--paper-soft:${c.paperSoft}`,
    `--ink:${c.ink}`,
    `--muted:${c.muted}`,
    `--faint:${c.faint}`,
    `--seam:${c.seam}`,
    `--seam-strong:${c.seamStrong}`,
    `--accent:${c.accent}`,
    `--accent-soft:${c.accentSoft}`,
    `--series-1:${c.series1}`,
    `--series-2:${c.series2}`,
    `--series-2-soft:${c.series2Soft}`,
    `--error:${c.error}`,
    `--warning:${c.warning}`,
    `--track:${c.track}`,
    `--outer-radius:${s.outer}px`,
    `--inner-radius:${s.inner}px`,
    `--control-radius:${s.control}px`,
    `--row-min:${s.row}px`,
    `--app-shadow:${s.shadow}`,
    `--glint:${s.glint}`,
    `--glow-opacity:${s.glowOpacity}`,
    `color-scheme:${theme.mode}`,
  ].join(";")
}
