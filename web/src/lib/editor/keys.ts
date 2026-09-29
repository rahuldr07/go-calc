export type KeyState = "live" | "held" | "act"
export type Key = readonly [label: string, insert: string, state: KeyState]

export const GROUP_NAMES = [
  ["sci", "sci"],
  ["calc", "calculus"],
  ["alg", "algebra"],
  ["mat", "matrix"],
  ["stat", "stats"],
  ["trust", "trust"],
] as const

export const PAD_GROUPS: Record<string, readonly Key[]> = {
  sci: [
    ["sin", "sin(", "live"], ["cos", "cos(", "live"], ["tan", "tan(", "live"], ["asin", "asin(", "live"],
    ["ln", "log(", "live"], ["log", "log10(", "live"], ["log₂", "log2(", "live"], ["exp", "exp(", "live"],
    ["√", "sqrt(", "live"], ["ⁿ√", "root(", "live"], ["x²", "^2", "live"], ["xʸ", "^", "live"],
    ["1/x", "1/", "live"], ["|x|", "abs(", "live"], ["n!", "!", "live"], ["mod", " mod ", "live"],
    ["π", "pi", "live"], ["e", "e", "live"], ["nCr", "binomial(", "held"], ["nPr", "perm(", "held"],
    ["EXP", "e", "live"], ["±", "±", "live"], ["sinh", "sinh(", "live"], ["atan", "atan(", "live"],
  ],
  calc: [
    ["d/dx", "diff(", "held"], ["d²/dx²", "diff2(", "held"], ["∂/∂x", "pdiff(", "held"], ["∇", "grad(", "held"],
    ["∫ dx", "integrate(", "held"], ["∫ᵇₐ", "integrate(", "held"], ["lim", "limit(", "held"], ["series", "series(", "held"],
    ["Σ", "sum(", "held"], ["Π", "prod(", "held"], ["taylor", "taylor(", "held"], ["ode", "ode(", "held"],
  ],
  alg: [
    ["solve", "solve(", "held"], ["roots", "roots(", "held"], ["factor", "factor(", "held"], ["expand", "expand(", "live"],
    ["simplify", "simplify(", "held"], ["collect", "collect(", "held"], ["apart", "apart(", "held"], ["subs", "subs(", "held"],
    ["gcd", "gcd(", "live"], ["lcm", "lcm(", "live"], ["isprime", "isprime(", "live"], ["divmod", "divmod(", "held"],
  ],
  mat: [
    ["[ ]", "[", "held"], ["det", "det(", "held"], ["inv", "inv(", "held"], ["Aᵀ", "transpose(", "held"],
    ["rank", "rank(", "held"], ["trace", "tr(", "held"], ["eig", "eigen(", "held"], ["A\\b", "lsolve(", "held"],
  ],
  stat: [
    ["mean", "mean(", "held"], ["median", "median(", "held"], ["sd", "sd(", "held"], ["var", "var(", "held"],
    ["sum", "total(", "held"], ["norm", "normal(", "held"], ["fit", "fit(", "held"], ["corr", "corr(", "held"],
  ],
  trust: [
    ["±", "±", "live"], ["corr", "corr(", "held"], ["k=2", "expand(ans, k=2)", "live"], ["digits", "digits ", "held"],
    ["trace", "trace ", "held"], ["why", "why", "held"], ["plot", "plot(", "held"], ["to", " to ", "live"],
  ],
}

export const CORE: readonly Key[] = [
  ["(", "(", "live"], [")", ")", "live"], ["del", "BACK", "live"], ["C", "CLEAR", "live"],
  ["7", "7", "live"], ["8", "8", "live"], ["9", "9", "live"], ["÷", "/", "live"],
  ["4", "4", "live"], ["5", "5", "live"], ["6", "6", "live"], ["×", "*", "live"],
  ["1", "1", "live"], ["2", "2", "live"], ["3", "3", "live"], ["−", "-", "live"],
  ["0", "0", "live"], [".", ".", "live"], ["%", "%", "live"], ["+", "+", "live"],
  ["ans", "ans", "live"], ["unit", " m", "live"], ["const", "pi", "live"], ["=", "EQ", "act"],
]

export const HELD_NOTE = "that engine is not built yet — phase 1 is the trust core"
