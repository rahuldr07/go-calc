export type KeyState = "live" | "held" | "act"
export type Key = readonly [label: string, insert: string, state: KeyState]

export const GROUP_NAMES = [
  ["sci", "sci"],
  ["calc", "calculus"],
  ["alg", "algebra"],
  ["stat", "stats"],
  ["unit", "units"],
] as const

export const PAD_GROUPS: Record<string, readonly Key[]> = {
  sci: [
    ["sin", "sin(", "live"], ["cos", "cos(", "live"], ["tan", "tan(", "live"], ["asin", "asin(", "live"],
    ["ln", "ln(", "live"], ["log", "log10(", "live"], ["log₂", "log2(", "live"], ["exp", "exp(", "live"],
    ["√", "sqrt(", "live"], ["ⁿ√", "root(", "live"], ["x²", "^2", "live"], ["xʸ", "^", "live"],
    ["1/x", "1/", "live"], ["|x|", "abs(", "live"], ["n!", "!", "live"], ["mod", " mod ", "live"],
    ["π", "pi", "live"], ["e", "e", "live"], ["i", "i", "live"], ["atan2", "atan2(", "live"],
    ["EXP", "e", "live"], ["±", " +- ", "live"], ["sinh", "sinh(", "live"], ["atan", "atan(", "live"],
  ],
  calc: [
    ["d/dx", "deriv(", "live"], ["x", "x", "live"], [",", ", ", "live"], ["solve", "solve(", "live"],
    ["Σ", "sum(", "live"], ["Π", "product(", "live"], ["∫ᵇₐ", "integrate(", "held"], ["lim", "limit(", "held"],
  ],
  alg: [
    ["solve", "solve(", "live"], ["x", "x", "live"], ["=", " = ", "live"], [",", ", ", "live"],
    ["<", " < ", "live"], [">", " > ", "live"], ["≤", " <= ", "live"], ["≥", " >= ", "live"],
    ["==", " == ", "live"], ["≠", " != ", "live"], ["gcd", "gcd(", "live"], ["lcm", "lcm(", "live"],
    ["isprime", "isprime(", "live"], ["floor", "floor(", "live"], ["ceil", "ceil(", "live"], ["round", "round(", "live"],
  ],
  stat: [
    ["[", "[", "live"], ["]", "]", "live"], [",", ", ", "live"], ["..", "..", "live"],
    ["sum", "sum(", "live"], ["mean", "mean(", "live"], ["median", "median(", "live"], ["count", "count(", "live"],
    ["min", "min(", "live"], ["max", "max(", "live"], ["stdev", "stdev(", "live"], ["variance", "variance(", "live"],
    ["stdevp", "stdevp(", "live"], ["product", "product(", "live"], ["±", " +- ", "live"], ["k=2", "expand(ans, k=2)", "live"],
  ],
  unit: [
    ["in", " in ", "live"], ["m", " m", "live"], ["km", " km", "live"], ["mile", " mile", "live"],
    ["kg", " kg", "live"], ["lb", " lb", "live"], ["s", " s", "live"], ["h", " h", "live"],
    ["L", " L", "live"], ["degC", " degC", "live"], ["degF", " degF", "live"], ["J", " J", "live"],
    ["c", "c", "live"], ["g0", "g0", "live"], ["N_A", "N_A", "live"], ["k_B", "k_B", "live"],
  ],
}

export const CORE: readonly Key[] = [
  ["(", "(", "live"], [")", ")", "live"], ["del", "BACK", "live"], ["C", "CLEAR", "live"],
  ["7", "7", "live"], ["8", "8", "live"], ["9", "9", "live"], ["÷", "/", "live"],
  ["4", "4", "live"], ["5", "5", "live"], ["6", "6", "live"], ["×", "*", "live"],
  ["1", "1", "live"], ["2", "2", "live"], ["3", "3", "live"], ["−", "-", "live"],
  ["0", "0", "live"], [".", ".", "live"], ["%", "%", "live"], ["+", "+", "live"],
  ["ans", "ans", "live"], ["in", " in ", "live"], ["^", "^", "live"], ["=", "EQ", "act"],
]

export const HELD_NOTE = "not built yet"
