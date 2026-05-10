import type { Reporter, TaskResultPack, File } from 'vitest'
import { readFileSync } from 'fs'
import { join } from 'path'

const B = '\x1b[1m'
const G = '\x1b[32m'
const R = '\x1b[31m'
const Y = '\x1b[33m'
const C = '\x1b[36m'
const D = '\x1b[90m'
const Z = '\x1b[0m'
const SEP = `${D}  ${'─'.repeat(62)}${Z}`

function bar(pct: number, width = 20): string {
  const filled = Math.round((pct / 100) * width)
  const col = pct >= 80 ? G : pct >= 60 ? Y : R
  return `${col}${'█'.repeat(filled)}${D}${'░'.repeat(width - filled)}${Z}`
}

function pctColor(pct: number): string {
  return pct >= 80 ? G : pct >= 60 ? Y : R
}

export default class TestCaseReporter implements Reporter {
  private taskMap = new Map<string, { name: string; type: string; suiteId?: string }>()
  private suiteNames = new Map<string, string>()
  private pass = 0
  private fail = 0
  private skip = 0
  private currentSuiteId = ''

  onCollected(files?: File[]) {
    files?.forEach(f => this.indexNode(f as any, undefined, 0))
  }

  private indexNode(task: any, nearestSuiteId: string | undefined, depth: number) {
    this.taskMap.set(task.id, { name: task.name ?? '', type: task.type ?? '', suiteId: nearestSuiteId })
    let newSuiteId = nearestSuiteId
    if (task.type === 'suite' && depth > 0) {
      newSuiteId = task.id
      this.suiteNames.set(task.id, task.name ?? '')
    }
    task.tasks?.forEach((t: any) => this.indexNode(t, newSuiteId, depth + 1))
  }

  onTaskUpdate(packs: TaskResultPack[]) {
    for (const [id, result] of packs) {
      if (!result || result.state === 'run' || result.state === 'queued') continue
      const task = this.taskMap.get(id)
      if (!task || task.type !== 'test') continue

      const state = result.state
      if (state === 'pass') this.pass++
      else if (state === 'fail') this.fail++
      else this.skip++

      // Print describe-block section header when group changes
      const suiteId = task.suiteId ?? ''
      if (suiteId !== this.currentSuiteId) {
        this.currentSuiteId = suiteId
        const suiteName = suiteId ? this.suiteNames.get(suiteId) ?? '' : ''
        if (suiteName) {
          process.stdout.write(`\n  ${C}${B}▶ ${suiteName}${Z}\n`)
        }
      }

      const name = task.name
      const m    = name.match(/([A-Z]+-UT-\d+)/)
      const tc   = m ? m[1] : null
      const raw  = tc ? name.slice(name.indexOf(tc) + tc.length) : name
      const desc = raw.replace(/^\s*[-–]\s*[✓✗]?\s*/, '').trim()

      const col   = state === 'pass' ? G : state === 'fail' ? R : D
      const sym   = state === 'pass' ? '✓' : state === 'fail' ? '✗' : '○'
      const badge = tc ? `${B}${col}[ ${tc} ]${Z}  ` : `${D}  `
      process.stdout.write(`${SEP}\n  ${badge}${col}${sym} ${desc}${Z}\n`)
    }
  }

  onFinished() {
    const total = this.pass + this.fail + this.skip
    const passRate = total > 0 ? (this.pass / total) * 100 : 0

    process.stdout.write(
      `\n${D}  ${'═'.repeat(62)}${Z}\n` +
      `  ${B}Kết quả:${Z}  ` +
      `${G}✓ ${this.pass} passed${Z}` +
      (this.fail ? `  ${R}✗ ${this.fail} failed${Z}` : '') +
      (this.skip ? `  ${D}○ ${this.skip} skipped${Z}` : '') +
      `  ${D}/ ${total} tổng${Z}\n`,
    )

    process.stdout.write(
      `\n  ${B}Pass rate ${Z} ${bar(passRate)}  ${pctColor(passRate)}${B}${passRate.toFixed(1)}%${Z}\n`,
    )

    this.printCoverage()
    process.stdout.write('\n')
  }

  private printCoverage() {
    try {
      const summaryPath = join(process.cwd(), 'coverage', 'coverage-summary.json')
      const raw = readFileSync(summaryPath, 'utf-8')
      const summary = JSON.parse(raw)
      const t = summary.total
      if (!t) return

      const metrics: [string, any][] = [
        ['Statements', t.statements],
        ['Branches  ', t.branches],
        ['Functions ', t.functions],
        ['Lines     ', t.lines],
      ]

      process.stdout.write(`\n  ${C}${B}Độ phủ (Coverage):${Z}\n`)
      for (const [label, m] of metrics) {
        const pct     = m?.pct     ?? 0
        const covered = m?.covered ?? 0
        const tot     = m?.total   ?? 0
        process.stdout.write(
          `  ${D}${label}${Z}  ${bar(pct)}  ${pctColor(pct)}${B}${pct.toFixed(1)}%${Z}` +
          `  ${D}(${covered}/${tot})${Z}\n`,
        )
      }
    } catch {
      // coverage-summary.json chưa có — bỏ qua
    }
  }
}
