import type { TraceNode } from "../engine/schema"

export interface StepInput {
  label: string
  value?: string
}

export interface Step {
  n: number
  id: number
  kind: string
  expression: string
  value?: string
  depth: number
  inputs: StepInput[]
  note?: string
}

const MAX_DEPTH = 32
const MAX_STEPS = 500
const ELLIPSIS = "…"

const WORK_KINDS = new Set(["operation", "function"])
const BINARY_PRECEDENCE: Record<string, number> = { "+": 1, "-": 1, "*": 2, "/": 2, "^": 4 }
const UNARY_PRECEDENCE = 3
const ATOM_PRECEDENCE = 5

interface Rendered {
  text: string
  precedence: number
}

function isWork(node: TraceNode): boolean {
  return WORK_KINDS.has(node.kind)
}

function isUnary(node: TraceNode): boolean {
  return node.kind === "operation" && node.label in BINARY_PRECEDENCE && node.children?.length === 1
}

function isBinary(node: TraceNode): boolean {
  return node.kind === "operation" && node.label in BINARY_PRECEDENCE && node.children?.length === 2
}

function indexById(nodes: TraceNode[]): Map<number, TraceNode> {
  const byId = new Map<number, TraceNode>()
  for (const node of nodes) {
    if (!byId.has(node.id)) byId.set(node.id, node)
  }
  return byId
}

function parenthesise(rendered: Rendered, needed: boolean): string {
  return needed ? `(${rendered.text})` : rendered.text
}

function render(
  node: TraceNode,
  byId: Map<number, TraceNode>,
  path: Set<number>,
  depth: number,
): Rendered {
  if (path.has(node.id) || depth > MAX_DEPTH) {
    return { text: ELLIPSIS, precedence: ATOM_PRECEDENCE }
  }
  const children = (node.children ?? [])
    .map((id) => byId.get(id))
    .filter((child): child is TraceNode => child !== undefined)
  if (children.length === 0 || !isWork(node)) {
    return { text: node.label, precedence: ATOM_PRECEDENCE }
  }

  path.add(node.id)
  const rendered = children.map((child) => render(child, byId, path, depth + 1))
  path.delete(node.id)

  if (isBinary(node) && rendered.length === 2) {
    const precedence = BINARY_PRECEDENCE[node.label]
    const rightAssociative = node.label === "^"
    const [left, right] = rendered
    const leftNeeds = rightAssociative ? left.precedence <= precedence : left.precedence < precedence
    const rightNeeds = rightAssociative ? right.precedence < precedence : right.precedence <= precedence
    const spacing = rightAssociative ? "" : " "
    return {
      text: `${parenthesise(left, leftNeeds)}${spacing}${node.label}${spacing}${parenthesise(right, rightNeeds)}`,
      precedence,
    }
  }

  if (isUnary(node) && rendered.length === 1) {
    const [operand] = rendered
    return {
      text: `${node.label}${parenthesise(operand, operand.precedence < UNARY_PRECEDENCE)}`,
      precedence: UNARY_PRECEDENCE,
    }
  }

  return {
    text: `${node.label}(${rendered.map((argument) => argument.text).join(", ")})`,
    precedence: ATOM_PRECEDENCE,
  }
}

function noteFor(node: TraceNode, children: TraceNode[]): string | undefined {
  const lines: string[] = []
  if (node.detail && node.detail !== node.label) lines.push(node.detail)
  for (const child of children) {
    if (!isWork(child) && child.detail && child.detail !== child.label) {
      lines.push(`${child.label}: ${child.detail}`)
    }
  }
  return lines.length > 0 ? lines.join("\n") : undefined
}

export function stepsOf(nodes: TraceNode[] | undefined): Step[] {
  if (!nodes || nodes.length === 0) return []
  const byId = indexById(nodes)
  const root = nodes[0]
  const emitted = new Set<number>()
  const steps: Step[] = []

  function visit(node: TraceNode, depth: number, path: Set<number>): void {
    if (steps.length >= MAX_STEPS || depth > MAX_DEPTH) return
    if (path.has(node.id) || emitted.has(node.id)) return
    if (!isWork(node)) return

    const children = (node.children ?? [])
      .map((id) => byId.get(id))
      .filter((child): child is TraceNode => child !== undefined)

    path.add(node.id)
    for (const child of children) visit(child, depth + 1, path)
    path.delete(node.id)

    if (steps.length >= MAX_STEPS || emitted.has(node.id)) return
    emitted.add(node.id)

    const step: Step = {
      n: steps.length + 1,
      id: node.id,
      kind: node.kind,
      expression: render(node, byId, new Set(), depth).text,
      depth,
      inputs: children.map((child) => ({
        label: isWork(child) ? render(child, byId, new Set([node.id]), depth + 1).text : child.label,
        ...(child.value !== undefined ? { value: child.value } : {}),
      })),
    }
    if (node.value !== undefined) step.value = node.value
    const note = noteFor(node, children)
    if (note !== undefined) step.note = note
    steps.push(step)
  }

  visit(root, 0, new Set())
  return steps
}
