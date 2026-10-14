export type ThemeId = "mineral" | "vellum" | "clearview" | "carbon" | "dimlight" | "void" | "blueprint" | "contrast"
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
  ok: string
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
      faint: "#5c6560",
      seam: "rgba(48,64,54,.15)",
      seamStrong: "rgba(48,64,54,.34)",
      accent: "#a14920",
      accentSoft: "#fbf1eb",
      series1: "#a14920",
      series2: "#2f6664",
      series2Soft: "#d7e6e2",
      error: "#a43d34",
      warning: "#8e591d",
      ok: "#2f6664",
      track: "#d9ded9",
    },
    shape: {
      outer: 24,
      inner: 19,
      control: 8,
      row: 46,
      shadow: "0 28px 84px rgba(37,52,43,.19), inset 0 1px rgba(255,255,255,.22)",
      glint: "radial-gradient(circle at 8% 0%, rgba(255,255,255,.72), transparent 26%)",
      glowOpacity: 0.34,
    },
  },
  vellum: {
    id: "vellum",
    label: "Vellum",
    note: "warm paper for long sessions",
    mode: "light",
    colors: {
      canvas: "#d8cdb4",
      shell: "#c5b898",
      paper: "#f5eddb",
      paperDeep: "#ebe1c9",
      paperSoft: "#f0e7d1",
      ink: "#2b2118",
      muted: "#5c4d3c",
      faint: "#6e5e4b",
      seam: "rgba(70,50,30,.14)",
      seamStrong: "rgba(70,50,30,.32)",
      accent: "#7a3566",
      accentSoft: "#eedbe4",
      series1: "#7a3566",
      series2: "#2f6b4a",
      series2Soft: "#d6e3d0",
      error: "#a1382a",
      warning: "#8a5a12",
      ok: "#2f6b4a",
      track: "#dccfb2",
    },
    shape: {
      outer: 20,
      inner: 15,
      control: 7,
      row: 48,
      shadow: "0 24px 70px rgba(70,50,30,.2), inset 0 1px rgba(255,250,235,.4)",
      glint: "radial-gradient(circle at 12% 0%, rgba(255,248,225,.6), transparent 30%)",
      glowOpacity: 0.2,
    },
  },
  clearview: {
    id: "clearview",
    label: "Clearview",
    note: "daylight, colour-blind safe",
    mode: "light",
    colors: {
      canvas: "#e4e7ea",
      shell: "#cfd4d9",
      paper: "#fdfdfb",
      paperDeep: "#f0f2f4",
      paperSoft: "#f7f8f9",
      ink: "#14181c",
      muted: "#475058",
      faint: "#566069",
      seam: "rgba(20,24,28,.14)",
      seamStrong: "rgba(20,24,28,.34)",
      accent: "#0a56a6",
      accentSoft: "#dbe8f6",
      series1: "#0a56a6",
      series2: "#a83200",
      series2Soft: "#f6e0d4",
      error: "#a83200",
      warning: "#7a5600",
      ok: "#00705a",
      track: "#d3d8dc",
    },
    shape: {
      outer: 12,
      inner: 8,
      control: 5,
      row: 48,
      shadow: "0 18px 50px rgba(20,24,28,.14)",
      glint: "none",
      glowOpacity: 0,
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
      faint: "#95a099",
      seam: "rgba(226,238,230,.09)",
      seamStrong: "rgba(226,238,230,.26)",
      accent: "#e79466",
      accentSoft: "#4a3024",
      series1: "#e79466",
      series2: "#67b8b7",
      series2Soft: "#263f3c",
      error: "#ed887f",
      warning: "#e4ad68",
      ok: "#67b8b7",
      track: "#353d38",
    },
    shape: {
      outer: 21,
      inner: 16,
      control: 7,
      row: 46,
      shadow: "0 30px 90px rgba(0,0,0,.26), inset 0 1px rgba(255,255,255,.07)",
      glint: "none",
      glowOpacity: 0.52,
    },
  },
  dimlight: {
    id: "dimlight",
    label: "Dimlight",
    note: "low-glare night shift",
    mode: "dark",
    colors: {
      canvas: "#0e0b09",
      shell: "#241d17",
      paper: "#1a1511",
      paperDeep: "#130f0c",
      paperSoft: "#1f1a15",
      ink: "#d8cbb8",
      muted: "#a99b87",
      faint: "#938676",
      seam: "rgba(216,203,184,.1)",
      seamStrong: "rgba(216,203,184,.24)",
      accent: "#b3a4e0",
      accentSoft: "#2e2742",
      series1: "#b3a4e0",
      series2: "#93c28d",
      series2Soft: "#26331f",
      error: "#f0937f",
      warning: "#d8b36a",
      ok: "#93c28d",
      track: "#2b241d",
    },
    shape: {
      outer: 22,
      inner: 17,
      control: 8,
      row: 46,
      shadow: "0 24px 70px rgba(0,0,0,.4), inset 0 1px rgba(255,235,210,.04)",
      glint: "radial-gradient(circle at 90% -10%, rgba(179,164,224,.1), transparent 38%)",
      glowOpacity: 0.22,
    },
  },
  void: {
    id: "void",
    label: "Void",
    note: "signal amber on black",
    mode: "dark",
    colors: {
      canvas: "#000000",
      shell: "#161615",
      paper: "#0c0c0b",
      paperDeep: "#080807",
      paperSoft: "#131312",
      ink: "#e6e2d6",
      muted: "#aaa699",
      faint: "#908c7f",
      seam: "rgba(230,226,214,.11)",
      seamStrong: "rgba(230,226,214,.26)",
      accent: "#ffb800",
      accentSoft: "#3b2d07",
      series1: "#ffb800",
      series2: "#e6e2d6",
      series2Soft: "#2a2822",
      error: "#ff5a5f",
      warning: "#ffc240",
      ok: "#8fcf9a",
      track: "#2a2a27",
    },
    shape: {
      outer: 22,
      inner: 16,
      control: 7,
      row: 46,
      shadow: "0 30px 100px rgba(0,0,0,.65), inset 0 1px rgba(255,255,255,.05)",
      glint: "radial-gradient(circle at 50% -24%, rgba(255,184,0,.12), transparent 46%)",
      glowOpacity: 0.4,
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
      paperDeep: "#dfe7f2",
      paperSoft: "#f6f8ff",
      ink: "#12213f",
      muted: "#4a5e7f",
      faint: "#51607c",
      seam: "rgba(26,59,116,.15)",
      seamStrong: "rgba(26,59,116,.34)",
      accent: "#2250cb",
      accentSoft: "#ceddff",
      series1: "#1f4cc2",
      series2: "#a53f35",
      series2Soft: "#f1dcd8",
      error: "#a53343",
      warning: "#76551c",
      ok: "#296c4f",
      track: "#cbd8ed",
    },
    shape: {
      outer: 16,
      inner: 12,
      control: 6,
      row: 44,
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
      canvas: "#1c1f24",
      shell: "#d9ddd9",
      paper: "#fbfcf8",
      paperDeep: "#f0f2ee",
      paperSoft: "#ffffff",
      ink: "#111411",
      muted: "#3d463f",
      faint: "#5a645d",
      seam: "rgba(17,20,17,.18)",
      seamStrong: "rgba(17,20,17,.46)",
      accent: "#064fc4",
      accentSoft: "#dce8ff",
      series1: "#064fc4",
      series2: "#8a245d",
      series2Soft: "#f3dfeb",
      error: "#9f1717",
      warning: "#7b4b00",
      ok: "#1a7437",
      track: "#c9ceca",
    },
    shape: {
      outer: 10,
      inner: 7,
      control: 4,
      row: 48,
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
    `--ok:${c.ok}`,
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
