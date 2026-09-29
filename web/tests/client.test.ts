import { describe, expect, it } from "vitest"
import { Engine } from "../src/lib/engine/client"
import { SUPERSEDED } from "../src/lib/engine/schema"

type Sent = { id: number; kind: string; request?: string }
type Handler = (message: Sent, worker: FakeWorker) => void

class FakeWorker {
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: ErrorEvent) => void) | null = null
  constructor(private readonly handle: Handler) {}
  postMessage(message: Sent) {
    queueMicrotask(() => this.handle(message, this))
  }
  reply(data: unknown) {
    this.onmessage?.({ data } as MessageEvent)
  }
  terminate() {}
}

function engineOver(handle: Handler) {
  const worker = new FakeWorker(handle)
  const engine = new Engine({ worker: worker as unknown as Worker })
  return { engine, worker }
}

describe("the engine client", () => {
  it("is ready once the module starts", async () => {
    let started: number | null = null
    const { engine } = engineOver((message, worker) => {
      started = message.id
      worker.reply({ id: message.id, ok: true })
    })
    await engine.ready
    expect(started).toBe(1)
    expect(engine.status).toBe("ready")
  })

  it("drops a response a later request has overtaken", async () => {
    const seen: number[] = []
    let fake!: FakeWorker
    const { engine } = engineOver((message, worker) => {
      fake = worker
      seen.push(message.id)
      if (message.kind === "start") worker.reply({ id: message.id, ok: true })
    })
    await engine.ready
    const slow = engine.evaluate({ lines: ["slow"], digits: 12, angle: "rad" })
    const fast = engine.evaluate({ lines: ["fast"], digits: 12, angle: "rad" })
    await Promise.resolve()
    fake.reply({ id: seen[2], ok: true, answer: '{"schema":"go-calc.sheet/1","lines":[{"text":"fast"}]}' })
    fake.reply({ id: seen[1], ok: true, answer: '{"schema":"go-calc.sheet/1","lines":[{"text":"slow"}]}' })
    expect((await fast).lines[0].text).toBe("fast")
    await expect(slow).rejects.toThrow(SUPERSEDED)
  })

  it("reports a module that will not load", async () => {
    const { engine } = engineOver((message, worker) => {
      if (message.kind === "start") worker.reply({ id: message.id, ok: false, error: "404" })
    })
    await expect(engine.ready).rejects.toThrow("404")
    expect(engine.status).toBe("failed")
  })

  it("refuses an answer that is not a sheet", async () => {
    const { engine } = engineOver((message, worker) => {
      if (message.kind === "start") worker.reply({ id: message.id, ok: true })
      if (message.kind === "evaluate") worker.reply({ id: message.id, ok: true, answer: "not json" })
    })
    await engine.ready
    await expect(engine.evaluate({ lines: ["1"], digits: 12, angle: "rad" })).rejects.toThrow("not a sheet")
  })
})
