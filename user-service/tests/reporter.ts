import type { Reporter, TestCase } from 'vitest/node'
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
  private pass = 0
  private fail = 0
  private skip = 0
  private currentSuite = ''

  onTestCaseResult(testCase: TestCase) {
    const result = testCase.result()
    const state = result.state
    if (state === 'pending') return

    if (state === 'passed') this.pass++
    else if (state === 'failed') this.fail++
    else this.skip++

    // Print describe-block header when group changes
    const suiteName = (testCase.parent as any)?.name ?? ''
    if (suiteName !== this.currentSuite) {
      this.currentSuite = suiteName
      if (suiteName) {
        process.stdout.write(`\n  ${C}${B}▶ ${suiteName}${Z}\n`)
      }
    }

    const name = testCase.name
    const m    = name.match(/([A-Z]+-UT-\d+)/)
    const tc   = m ? m[1] : null
    const raw  = tc ? name.slice(name.indexOf(tc) + tc.length) : name
    const desc = raw.replace(/^\s*[-–]\s*[✓✗]?\s*/, '').trim()

    const col   = state === 'passed' ? G : state === 'failed' ? R : D
    const sym   = state === 'passed' ? '✓' : state === 'failed' ? '✗' : '○'
    const badge = tc ? `${B}${col}[ ${tc} ]${Z}  ` : `${D}  `
    process.stdout.write(`${SEP}\n  ${badge}${col}${sym} ${desc}${Z}\n`)
  }

  onTestRunEnd() {
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
