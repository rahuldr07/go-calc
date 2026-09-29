export type Status = "exact" | "proven" | "estimate" | "held" | "error"
export type NoteSource = "parser" | "evaluator"
export type Angle = "rad" | "deg"

export interface Note {
  source: NoteSource
  text: string
  column?: number
}

export interface Contribution {
  name: string
  share: number
  sigma: number
  sensitivity: number
}

export interface Uncertainty {
  value?: string
  reported?: string
  standard?: string
  coverage?: string
  factor?: number
  budget?: Contribution[]
  nonlinear?: string
}

export interface TraceNode {
  id: number
  kind: string
  label: string
  detail?: string
  children?: number[]
}

export interface Line {
  n: number
  text: string
  qualifiers?: string[]
  status: Status
  digits: number
  capped?: boolean
  stopped?: string
  error?: string
  column?: number
  uncertainty?: Uncertainty
  notes?: Note[]
  warning?: string
  trace?: TraceNode[]
}

export interface Sheet {
  schema: string
  lines: Line[]
}

export interface Request {
  lines: string[]
  digits: number
  angle: Angle
}

export const SUPERSEDED = "superseded"
