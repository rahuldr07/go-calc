const SUPERSCRIPT: Record<string, string> = {
  "0": "⁰", "1": "¹", "2": "²", "3": "³", "4": "⁴",
  "5": "⁵", "6": "⁶", "7": "⁷", "8": "⁸", "9": "⁹", "-": "⁻",
}

const DIMENSION_ORDER = ["L", "M", "T", "I", "Θ", "K", "N", "J"]

const DIMENSION_NAMES: Record<string, string> = {
  L: "length",
  M: "mass",
  T: "time",
  I: "current",
  Θ: "temperature",
  K: "temperature",
  N: "amount",
  J: "luminosity",
}

const LEAD_DIGITS = 15
const DIGIT_LIMIT_FOR_VALUE = 15

export interface DimensionTerm {
  symbol: string
  name: string
  exponent: number
}

export interface LogReading {
  sign: -1 | 0 | 1
  log10: number
  wholeDigits: number | null
}

export interface Magnitude extends LogReading {
  isInteger: boolean
}

export interface Scientific {
  mantissa: string
  exponent: number
}

export interface Tick {
  value: number
  isReference: boolean
}

export interface RatioReading {
  log10: number
  fraction: number
  numeratorIsLarger: boolean
  numeratorIsZero: boolean
}

export interface ShareRow {
  name: string
  share: number
  sigma: number
  sensitivity: number
}

export function superscript(text: string): string {
  return text.split("").map((c) => SUPERSCRIPT[c] ?? c).join("")
}

export function exponentLabel(symbol: string, exponent: number): string {
  if (exponent === 1) return symbol
  if (Number.isInteger(exponent)) return symbol + superscript(String(exponent))
  return `${symbol}^${exponent}`
}

export function orderedDimensions(dims: Record<string, number> | undefined): DimensionTerm[] {
  if (!dims) return []
  const rank = (symbol: string): number => {
    const at = DIMENSION_ORDER.indexOf(symbol)
    return at === -1 ? DIMENSION_ORDER.length : at
  }
  return Object.entries(dims)
    .filter(([, exponent]) => Number.isFinite(exponent) && exponent !== 0)
    .map(([symbol, exponent]) => ({ symbol, name: DIMENSION_NAMES[symbol] ?? symbol, exponent }))
    .sort((a, b) => rank(a.symbol) - rank(b.symbol) || a.symbol.localeCompare(b.symbol))
}

export function dimensionFormula(terms: DimensionTerm[]): string {
  const above = terms.filter((t) => t.exponent > 0).map((t) => exponentLabel(t.symbol, t.exponent))
  const below = terms.filter((t) => t.exponent < 0).map((t) => exponentLabel(t.symbol, -t.exponent))
  if (above.length === 0 && below.length === 0) return "dimensionless"
  const top = above.length > 0 ? above.join(" ") : "1"
  return below.length > 0 ? `${top} / ${below.join(" ")}` : top
}

export function digitCount(integer: string): number {
  const digits = integer.replace(/^[+-]/, "").replace(/^0+/, "")
  return digits.length
}

export function groupDigits(integer: string): string {
  const negative = integer.startsWith("-")
  const body = integer.replace(/^[+-]/, "")
  const grouped = body.replace(/\B(?=(\d{3})+(?!\d))/g, ",")
  return negative ? "-" + grouped : grouped
}

export function logOfDecimal(text: string): LogReading | null {
  const match = /^([+-]?)(\d*)\.?(\d*)(?:[eE]([+-]?\d+))?$/.exec(text.trim())
  if (!match) return null
  const [, signText, whole, fraction, exponentText] = match
  if (whole === "" && fraction === "") return null
  const digits = whole + fraction
  const firstSignificant = digits.search(/[1-9]/)
  if (firstSignificant === -1) return { sign: 0, log10: Number.NEGATIVE_INFINITY, wholeDigits: null }
  const pointAt = whole.length + (exponentText ? Number(exponentText) : 0)
  const lead = digits.slice(firstSignificant, firstSignificant + LEAD_DIGITS)
  const leadLog = Math.log10(Number(lead) / 10 ** (lead.length - 1))
  const exponent = pointAt - firstSignificant - 1
  const integerDigits = fraction === "" && !exponentText ? pointAt - firstSignificant : null
  return {
    sign: signText === "-" ? -1 : 1,
    log10: exponent + leadLog,
    wholeDigits: integerDigits !== null && integerDigits > 0 ? integerDigits : null,
  }
}

export function splitRational(exact: string): { numerator: string; denominator: string } | null {
  const parts = exact.trim().split("/")
  if (parts.length > 2) return null
  const numerator = parts[0]
  const denominator = parts[1] ?? "1"
  if (!/^[+-]?\d+$/.test(numerator) || !/^\d+$/.test(denominator)) return null
  if (digitCount(denominator) === 0) return null
  return { numerator, denominator }
}

export function magnitudeOf(input: { exact?: string; decimal?: string }): Magnitude | null {
  if (input.exact) {
    const parts = splitRational(input.exact)
    if (parts) {
      const top = logOfDecimal(parts.numerator)
      const bottom = logOfDecimal(parts.denominator)
      if (top && bottom) {
        const isInteger = digitCount(parts.denominator) === 1 && parts.denominator.replace(/^0+/, "") === "1"
        return {
          sign: top.sign,
          log10: top.sign === 0 ? Number.NEGATIVE_INFINITY : top.log10 - bottom.log10,
          wholeDigits: isInteger ? top.wholeDigits : null,
          isInteger,
        }
      }
    }
  }
  if (input.decimal) {
    const reading = logOfDecimal(input.decimal)
    if (reading) return { ...reading, isInteger: reading.wholeDigits !== null }
  }
  return null
}

export function scientific(log10: number): Scientific {
  let exponent = Math.floor(log10)
  let mantissa = 10 ** (log10 - exponent)
  if (mantissa >= 9.995) {
    mantissa = 1
    exponent += 1
  }
  return { mantissa: mantissa.toFixed(2), exponent }
}

export function scientificLabel(log10: number): string {
  const { mantissa, exponent } = scientific(log10)
  return `${mantissa} × 10${superscript(String(exponent))}`
}

export function headlineFor(magnitude: Magnitude): string {
  const sign = magnitude.sign < 0 ? "−" : ""
  if (magnitude.isInteger && magnitude.wholeDigits !== null && magnitude.wholeDigits > DIGIT_LIMIT_FOR_VALUE) {
    return `${sign}${groupDigits(String(magnitude.wholeDigits))} digits`
  }
  return `≈ ${sign}${scientificLabel(magnitude.log10)}`
}

export function niceStep(span: number, target: number): number {
  const raw = span / Math.max(1, target)
  const power = Math.floor(Math.log10(raw))
  const unit = 10 ** power
  const scaled = raw / unit
  const choice = scaled <= 1 ? 1 : scaled <= 2 ? 2 : scaled <= 5 ? 5 : 10
  return Math.max(1, choice * unit)
}

export function niceTicks(lo: number, hi: number, target: number): Tick[] {
  const step = niceStep(hi - lo, target)
  const first = Math.ceil(lo / step) * step
  const ticks: Tick[] = []
  for (let value = first; value <= hi + 1e-9; value += step) {
    const rounded = Math.round(value)
    ticks.push({ value: rounded, isReference: rounded === 0 })
  }
  return ticks
}

export function logDomain(log10: number, minimumSpan: number): [number, number] {
  const low = Math.min(log10, 0)
  const high = Math.max(log10, 0)
  const pad = Math.max(1, (high - low) * 0.06)
  let start = low - pad
  let end = high + pad
  const missing = minimumSpan - (end - start)
  if (missing > 0) {
    start -= missing / 2
    end += missing / 2
  }
  return [Math.floor(start), Math.ceil(end)]
}

export function placeOn(value: number, lo: number, hi: number, x0: number, x1: number): number {
  const t = (value - lo) / (hi - lo)
  return x0 + Math.min(1, Math.max(0, t)) * (x1 - x0)
}

export function tickLabel(exponent: number): string {
  return exponent === 0 ? "1" : "10" + superscript(String(exponent))
}

export function ratioOf(numerator: string, denominator: string): RatioReading | null {
  const top = logOfDecimal(numerator)
  const bottom = logOfDecimal(denominator)
  if (!top || !bottom || bottom.sign === 0) return null
  if (top.sign === 0) {
    return { log10: Number.NEGATIVE_INFINITY, fraction: 0, numeratorIsLarger: false, numeratorIsZero: true }
  }
  const log10 = top.log10 - bottom.log10
  const numeratorIsLarger = log10 > 0
  const fraction = 10 ** (numeratorIsLarger ? -log10 : log10)
  return { log10, fraction, numeratorIsLarger, numeratorIsZero: false }
}

export function ratioSentence(reading: RatioReading): string {
  if (reading.numeratorIsZero) return "the numerator is zero, so the value is zero"
  if (reading.numeratorIsLarger) {
    return `the numerator is ≈ ${scientificLabel(reading.log10)} times the denominator`
  }
  if (reading.log10 > -4) {
    const percent = 10 ** (reading.log10 + 2)
    const digits = percent >= 10 ? 3 : 2
    return `the numerator is ≈ ${Number(percent.toPrecision(digits))}% of the denominator`
  }
  return `the numerator is ≈ ${scientificLabel(reading.log10)} of the denominator`
}

export function integerLabel(integer: string): string {
  const count = digitCount(integer)
  if (count > 12) return `${groupDigits(String(count))} digits`
  return groupDigits(integer)
}

export function normalizeShares(rows: ShareRow[]): ShareRow[] {
  const total = rows.reduce((sum, r) => sum + Math.max(0, r.share), 0)
  const ranked = rows
    .map((r) => ({ ...r, share: total > 0 ? Math.max(0, r.share) / total : 0 }))
    .sort((a, b) => b.share - a.share)
  return ranked
}

export function shareLabel(share: number): string {
  const percent = share * 100
  if (percent === 0) return "0%"
  if (percent < 0.1) return "<0.1%"
  if (percent >= 99.95) return ">99.9%"
  return `${percent.toFixed(percent >= 10 ? 1 : 2)}%`
}

export function clipName(name: string, limit: number): string {
  return name.length <= limit ? name : name.slice(0, limit - 1) + "…"
}
