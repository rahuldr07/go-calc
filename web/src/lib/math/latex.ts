type TokenKind = "number" | "ident" | "op"

interface Token {
  kind: TokenKind
  text: string
  start: number
  end: number
}

interface Piece {
  tex: string
  tall: boolean
  atomic: boolean
  numeric: boolean
  inner?: string
}

class ParseError extends Error {}

const GREEK: Record<string, string> = {
  pi: "\\pi",
  alpha: "\\alpha",
  beta: "\\beta",
  gamma: "\\gamma",
  delta: "\\delta",
  epsilon: "\\epsilon",
  zeta: "\\zeta",
  eta: "\\eta",
  theta: "\\theta",
  kappa: "\\kappa",
  lambda: "\\lambda",
  mu: "\\mu",
  nu: "\\nu",
  xi: "\\xi",
  rho: "\\rho",
  sigma: "\\sigma",
  tau: "\\tau",
  phi: "\\phi",
  chi: "\\chi",
  psi: "\\psi",
  omega: "\\omega",
  hbar: "\\hbar",
  inf: "\\infty",
}

const FUNCTION_NAMES: Record<string, string> = {
  sin: "\\sin",
  cos: "\\cos",
  tan: "\\tan",
  sec: "\\sec",
  csc: "\\csc",
  cot: "\\cot",
  sinh: "\\sinh",
  cosh: "\\cosh",
  tanh: "\\tanh",
  asin: "\\arcsin",
  acos: "\\arccos",
  atan: "\\arctan",
  ln: "\\ln",
  log: "\\log",
  log2: "\\log_{2}",
  log10: "\\log_{10}",
  exp: "\\exp",
  min: "\\min",
  max: "\\max",
  gcd: "\\gcd",
}

const BRACKETED_FUNCTIONS: Record<string, [string, string]> = {
  abs: ["\\left|", "\\right|"],
  floor: ["\\left\\lfloor ", "\\right\\rfloor "],
  ceil: ["\\left\\lceil ", "\\right\\rceil "],
}

const SYMBOL_ALIASES: Record<string, string> = {
  "×": "*",
  "·": "*",
  "÷": "/",
  "−": "-",
  "±": "+-",
  "π": "pi",
}

const TEXT_ESCAPES: Record<string, string> = {
  "\\": "\\textbackslash{}",
  "&": "\\&",
  "%": "\\%",
  $: "\\$",
  "#": "\\#",
  _: "\\_",
  "{": "\\{",
  "}": "\\}",
  "~": "\\textasciitilde{}",
  "^": "\\textasciicircum{}",
}

function lookup<Value>(table: Record<string, Value>, key: string): Value | undefined {
  return Object.hasOwn(table, key) ? table[key] : undefined
}

const NUMBER_PATTERN = /(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?/y
const IDENT_PATTERN = /[A-Za-z_][A-Za-z0-9_]*/y
const OPERATORS = ["+-", "**", "+", "-", "*", "/", "^", "(", ")", ",", "!", "%"]

function normalize(source: string): string {
  return Array.from(source, (character) => SYMBOL_ALIASES[character] ?? character).join("")
}

function tokenize(source: string): Token[] {
  const tokens: Token[] = []
  let position = 0
  while (position < source.length) {
    if (/\s/.test(source[position])) {
      position += 1
      continue
    }
    NUMBER_PATTERN.lastIndex = position
    const number = NUMBER_PATTERN.exec(source)
    if (number) {
      tokens.push({ kind: "number", text: number[0], start: position, end: position + number[0].length })
      position += number[0].length
      continue
    }
    IDENT_PATTERN.lastIndex = position
    const ident = IDENT_PATTERN.exec(source)
    if (ident) {
      tokens.push({ kind: "ident", text: ident[0], start: position, end: position + ident[0].length })
      position += ident[0].length
      continue
    }
    const operator = OPERATORS.find((candidate) => source.startsWith(candidate, position))
    if (!operator) throw new ParseError(`unexpected character at ${position}`)
    const text = operator === "**" ? "^" : operator
    tokens.push({ kind: "op", text, start: position, end: position + operator.length })
    position += operator.length
  }
  return tokens
}

function numberTex(text: string): string {
  const match = /^(.*?)[eE]([+-]?\d+)$/.exec(text)
  if (!match) return text
  return `${match[1]}\\times10^{${match[2]}}`
}

function subscriptTex(subscript: string): string {
  if (subscript.length === 1 || /^\d+$/.test(subscript)) return subscript
  return `\\mathrm{${subscript.replace(/_/g, "\\_")}}`
}

function identifierTex(name: string, asUnit: boolean): string {
  const separator = name.indexOf("_")
  const base = separator === -1 ? name : name.slice(0, separator)
  const subscript = separator === -1 ? "" : name.slice(separator + 1)
  if (base === "" || (separator !== -1 && subscript === "")) throw new ParseError("dangling underscore")
  const greek = lookup(GREEK, base)
  let baseTex: string
  if (greek) baseTex = greek
  else if (asUnit || base.length > 1) baseTex = `\\mathrm{${base}}`
  else baseTex = base
  return subscript ? `${baseTex}_{${subscriptTex(subscript)}}` : baseTex
}

function wrapParentheses(inner: string, tall: boolean): string {
  return tall ? `\\left(${inner}\\right)` : `(${inner})`
}

class Parser {
  private index = 0

  constructor(private readonly tokens: Token[]) {}

  parse(): string {
    const result = this.parseSum()
    if (this.index < this.tokens.length) throw new ParseError("trailing input")
    return result.tex
  }

  private peek(): Token | undefined {
    return this.tokens[this.index]
  }

  private peekOperator(...texts: string[]): string | null {
    const token = this.peek()
    if (token && token.kind === "op" && texts.includes(token.text)) return token.text
    return null
  }

  private expectOperator(text: string): void {
    if (!this.peekOperator(text)) throw new ParseError(`expected ${text}`)
    this.index += 1
  }

  private startsFactor(): boolean {
    const token = this.peek()
    if (!token) return false
    return token.kind !== "op" || token.text === "("
  }

  private parseSum(): Piece {
    const first = this.parseTerm()
    let tex = first.tex
    let tall = first.tall
    let terms = 1
    for (let joiner = this.peekOperator("+", "-", "+-"); joiner; joiner = this.peekOperator("+", "-", "+-")) {
      this.index += 1
      const next = this.parseTerm()
      tex += joiner === "+-" ? "\\pm " : joiner
      tex += next.tex
      tall = tall || next.tall
      terms += 1
    }
    return terms === 1 ? first : { tex, tall, atomic: false, numeric: false }
  }

  private parseTerm(): Piece {
    let head = ""
    let last: Piece = this.parseFactor(false)
    let tall = last.tall
    let lastIsFraction = false
    let unitRun = false
    while (true) {
      if (this.peekOperator("*")) {
        this.index += 1
        const next = this.parseFactor(false)
        head += `${last.tex}\\cdot `
        last = next
        unitRun = false
        lastIsFraction = false
      } else if (this.peekOperator("/")) {
        this.index += 1
        const next = this.parseFactor(unitRun)
        if (last.atomic && next.atomic && !lastIsFraction) {
          last = {
            tex: `\\frac{${last.inner ?? last.tex}}{${next.inner ?? next.tex}}`,
            tall: true,
            atomic: false,
            numeric: false,
          }
          lastIsFraction = true
        } else {
          head += `${last.tex}/`
          last = next
          lastIsFraction = false
        }
      } else if (this.startsFactor()) {
        const token = this.peek() as Token
        const isConstant = token.kind === "ident" && lookup(GREEK, token.text) !== undefined
        const asUnit: boolean = token.kind === "ident" && !isConstant && (last.numeric || unitRun)
        const next = this.parseFactor(asUnit)
        const tight = asUnit || (isConstant && last.numeric)
        head += `${last.tex}${tight ? "\\," : "\\cdot "}`
        last = next
        unitRun = asUnit
        lastIsFraction = false
      } else {
        break
      }
      tall = tall || last.tall
    }
    if (head === "") return last
    return { tex: head + last.tex, tall, atomic: false, numeric: false }
  }

  private parseFactor(asUnit: boolean): Piece {
    const sign = this.peekOperator("-", "+")
    if (!sign) return this.parsePower(asUnit)
    this.index += 1
    const operand = this.parsePower(asUnit)
    return { tex: `${sign}${operand.tex}`, tall: operand.tall, atomic: false, numeric: false }
  }

  private parsePower(asUnit: boolean): Piece {
    const base = this.parsePostfix(asUnit)
    if (!this.peekOperator("^")) return base
    this.index += 1
    const exponent = this.parseExponent()
    return {
      tex: `${base.tex}^{${exponent.tex}}`,
      tall: base.tall || exponent.tall,
      atomic: true,
      numeric: false,
    }
  }

  private parseExponent(): Piece {
    const sign = this.peekOperator("-", "+")
    if (sign) this.index += 1
    const operand = this.parsePower(false)
    return { ...operand, tex: `${sign ?? ""}${operand.inner ?? operand.tex}` }
  }

  private parsePostfix(asUnit: boolean): Piece {
    let piece = this.parseAtom(asUnit)
    for (let mark = this.peekOperator("!", "%"); mark; mark = this.peekOperator("!", "%")) {
      this.index += 1
      piece = { tex: `${piece.tex}${mark === "%" ? "\\%" : "!"}`, tall: piece.tall, atomic: false, numeric: false }
    }
    return piece
  }

  private parseAtom(asUnit: boolean): Piece {
    const token = this.peek()
    if (!token) throw new ParseError("unexpected end")
    if (token.kind === "number") {
      this.index += 1
      return { tex: numberTex(token.text), tall: false, atomic: true, numeric: true }
    }
    if (token.kind === "ident") {
      this.index += 1
      const next = this.peek()
      const isCall = next && next.kind === "op" && next.text === "(" && next.start === token.end
      if (isCall) return this.parseCall(token.text)
      return { tex: identifierTex(token.text, asUnit), tall: false, atomic: true, numeric: false }
    }
    if (token.text === "(") {
      this.index += 1
      const inner = this.parseSum()
      this.expectOperator(")")
      return {
        tex: wrapParentheses(inner.tex, inner.tall),
        tall: inner.tall,
        atomic: true,
        numeric: false,
        inner: inner.tex,
      }
    }
    throw new ParseError(`unexpected ${token.text}`)
  }

  private parseArguments(): Piece[] {
    this.expectOperator("(")
    const pieces = [this.parseSum()]
    while (this.peekOperator(",")) {
      this.index += 1
      pieces.push(this.parseSum())
    }
    this.expectOperator(")")
    return pieces
  }

  private parseCall(name: string): Piece {
    const pieces = this.parseArguments()
    const joined = pieces.map((piece) => piece.tex).join(",\\,")
    const tall = pieces.some((piece) => piece.tall)
    const single = pieces.length === 1
    if (name === "sqrt" && single) return { tex: `\\sqrt{${joined}}`, tall: true, atomic: true, numeric: false }
    if (name === "cbrt" && single) return { tex: `\\sqrt[3]{${joined}}`, tall: true, atomic: true, numeric: false }
    const bracketed = lookup(BRACKETED_FUNCTIONS, name)
    if (bracketed && single) {
      return { tex: `${bracketed[0]}${joined}${bracketed[1]}`, tall: false, atomic: true, numeric: false }
    }
    const known = lookup(FUNCTION_NAMES, name)
    const label = known ?? identifierTex(name, true)
    return { tex: `${label}${wrapParentheses(joined, tall)}`, tall, atomic: true, numeric: false }
  }
}

function escapeForTextMode(source: string): string {
  const escaped = Array.from(source, (character) => TEXT_ESCAPES[character] ?? character).join("")
  return `\\text{${escaped}}`
}

export function toLatex(source: string): string {
  try {
    const tokens = tokenize(normalize(source))
    if (tokens.length === 0) return ""
    return new Parser(tokens).parse().trim()
  } catch {
    return escapeForTextMode(source)
  }
}
