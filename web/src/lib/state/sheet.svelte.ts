import { Engine, type EngineStatus } from "../engine/client"
import { SUPERSEDED, type Angle, type Line } from "../engine/schema"

const engine = new Engine()

const DOCUMENT = "gocalc-document"
const DIGITS = "gocalc-digits"

function saved(key: string): string {
  try {
    return localStorage.getItem(key) ?? ""
  } catch {
    return ""
  }
}

function restored(): string {
  return saved(DOCUMENT)
}

function keep(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    return
  }
}

export const sheet = $state({
  source: restored(),
  lines: [] as Line[],
  digits: Number(saved(DIGITS)) || 12,
  angle: "rad" as Angle,
  engineStatus: "loading" as EngineStatus,
  failure: "",
  elapsed: 0,
  selected: null as number | null,
})

engine.ready.then(
  () => {
    sheet.engineStatus = "ready"
    void recompute()
  },
  (err: Error) => {
    sheet.engineStatus = "failed"
    sheet.failure = err.message
  },
)

export function lineAt(n: number): Line | undefined {
  return sheet.lines.find((line) => line.n === n)
}

export function selectedLine(): Line | undefined {
  return sheet.selected === null ? undefined : lineAt(sheet.selected)
}

export async function setSource(text: string) {
  sheet.source = text
  keep(DOCUMENT, text)
  await recompute()
}

export async function setDigits(digits: number) {
  sheet.digits = Math.min(1000, Math.max(1, Math.round(digits)))
  keep(DIGITS, String(sheet.digits))
  await recompute()
}

export async function setAngle(angle: Angle) {
  sheet.angle = angle
  await recompute()
}

export function select(n: number | null) {
  sheet.selected = n
}

async function recompute() {
  if (sheet.engineStatus !== "ready") return
  const started = performance.now()
  try {
    const answer = await engine.evaluate({
      lines: sheet.source.split("\n"),
      digits: sheet.digits,
      angle: sheet.angle,
    })
    sheet.lines = answer.lines
    sheet.elapsed = Math.round(performance.now() - started)
  } catch (err) {
    if (err instanceof Error && err.message === SUPERSEDED) return
    sheet.engineStatus = "failed"
    sheet.failure = err instanceof Error ? err.message : String(err)
  }
}
