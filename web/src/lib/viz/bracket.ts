import type { Line } from "../engine/schema"

export type ReadingKind = "proven" | "exact" | "measured" | "none"

export interface Reading {
  kind: ReadingKind
  shown: string
  certain: string
  loTail: string
  hiTail: string
  lo?: string
  hi?: string
  exact?: string
  exactIsWhole: boolean
  width?: string
  unit?: string
  dims?: string
  digits: number
  capped: boolean
  sigma?: number
  reported?: string
}

const SUPER: Record<string, string> = {
  "0": "⁰", "1": "¹", "2": "²", "3": "³", "4": "⁴",
  "5": "⁵", "6": "⁶", "7": "⁷", "8": "⁸", "9": "⁹", "-": "⁻",
}

export function dimsLabel(dims: Record<string, number> | undefined): string {
  if (!dims) return ""
  return Object.entries(dims)
    .filter(([, e]) => e !== 0)
    .map(([sym, e]) => (e === 1 ? sym : sym + String(e).split("").map((c) => SUPER[c] ?? c).join("")))
    .join(" ")
}

export function parseBall(ball: string | undefined): [string, string] | null {
  if (!ball) return null
  const inner = ball.trim().replace(/^\[/, "").replace(/\]$/, "")
  const parts = inner.split(",").map((p) => p.trim())
  if (parts.length !== 2 || !parts[0] || !parts[1]) return null
  return [parts[0], parts[1]]
}

function sharedPrefix(a: string, b: string): number {
  const n = Math.min(a.length, b.length)
  let i = 0
  while (i < n && a[i] === b[i]) i++
  return i
}

function widthLabel(lo: string, hi: string): string | undefined {
  const a = Number(lo)
  const b = Number(hi)
  if (!Number.isFinite(a) || !Number.isFinite(b)) return undefined
  const w = Math.abs(b - a)
  if (w === 0) return "0"
  return w.toExponential(1)
}

export function readingOf(line: Line | undefined): Reading | null {
  if (!line || line.status === "error" || line.status === "held") return null
  const detail = line.value
  if (!detail) return null

  const shown = detail.decimal ?? line.text
  const base: Reading = {
    kind: "none",
    shown,
    certain: shown,
    loTail: "",
    hiTail: "",
    exactIsWhole: false,
    digits: line.digits,
    capped: line.capped ?? false,
    unit: line.unit?.symbol,
    dims: dimsLabel(line.unit?.dims),
    sigma: line.uncertainty?.standard,
    reported: line.uncertainty?.reported,
  }

  const bounds = parseBall(detail.ball)
  if (bounds) {
    const [lo, hi] = bounds
    const cut = sharedPrefix(lo, hi)
    return {
      ...base,
      kind: "proven",
      lo,
      hi,
      certain: lo.slice(0, cut),
      loTail: lo.slice(cut),
      hiTail: hi.slice(cut),
      width: widthLabel(lo, hi),
    }
  }

  if (detail.exact) {
    const measured = line.status === "estimate"
    const value = measured ? (line.uncertainty?.value ?? shown) : shown
    return {
      ...base,
      kind: measured ? "measured" : "exact",
      shown: value,
      certain: value,
      exact: detail.exact,
      exactIsWhole: !detail.exact.includes("/"),
      width: line.uncertainty?.standard ? undefined : "0",
    }
  }

  return base
}
