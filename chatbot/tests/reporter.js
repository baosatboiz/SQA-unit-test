// Jest custom reporter — in kết quả mỗi test với màu xanh/đỏ, nhóm theo describe block, hiển thị coverage
const { readFileSync } = require('fs')
const { join } = require('path')

const B = '\x1b[1m', G = '\x1b[32m', R = '\x1b[31m', Y = '\x1b[33m', C = '\x1b[36m', D = '\x1b[90m', Z = '\x1b[0m'
const SEP = `${D}  ${'─'.repeat(62)}${Z}`

function bar(pct, width = 20) {
  const filled = Math.round((pct / 100) * width)
  const col = pct >= 80 ? G : pct >= 60 ? Y : R
  return `${col}${'█'.repeat(filled)}${D}${'░'.repeat(width - filled)}${Z}`
}

function pctColor(pct) {
  return pct >= 80 ? G : pct >= 60 ? Y : R
}

class TestCaseReporter {
  constructor() {
    this.currentSuite = ''
  }

  onRunStart() {
    this.currentSuite = ''
  }

  onTestFileResult(_test, fileResult) {
    // File name header
    const fileName = fileResult.testFilePath.replace(/\\/g, '/').split('/').pop()
    process.stdout.write(`\n  ${C}${B}━━ ${fileName} ━━${Z}\n`)
    this.currentSuite = ''

    for (const result of fileResult.testResults) {
      const name   = result.title
      const status = result.status
      const ancestorTitles = result.ancestorTitles ?? []

      // Print describe-block header when group changes
      const suiteName = ancestorTitles.length > 0 ? ancestorTitles[ancestorTitles.length - 1] : ''
      if (suiteName !== this.currentSuite) {
        this.currentSuite = suiteName
        if (suiteName) {
          process.stdout.write(`\n  ${C}▶ ${suiteName}${Z}\n`)
        }
      }

      const m    = name.match(/([A-Z]+-UT-\d+)/)
      const id   = m ? m[1] : null
      const raw  = id ? name.slice(name.indexOf(id) + id.length) : name
      const desc = raw.replace(/^\s*[-–]\s*[✓✗]?\s*/, '').trim()

      const col   = status === 'passed' ? G : status === 'failed' ? R : D
      const sym   = status === 'passed' ? '✓' : status === 'failed' ? '✗' : '○'
      const badge = id ? `${B}${col}[ ${id} ]${Z}  ` : ''

      process.stdout.write(`${SEP}\n  ${badge}${col}${sym} ${desc}${Z}\n`)
    }
  }

  onRunComplete(_contexts, results) {
    const { numPassedTests, numFailedTests, numPendingTests } = results
    const total = numPassedTests + numFailedTests + numPendingTests
    const passRate = total > 0 ? (numPassedTests / total) * 100 : 0

    process.stdout.write(
      `\n${D}  ${'═'.repeat(62)}${Z}\n` +
      `  ${B}Kết quả:${Z}  ` +
      `${G}✓ ${numPassedTests} passed${Z}` +
      (numFailedTests  ? `  ${R}✗ ${numFailedTests} failed${Z}` : '') +
      (numPendingTests ? `  ${D}○ ${numPendingTests} skipped${Z}` : '') +
      `  ${D}/ ${total} tổng${Z}\n`
    )

    process.stdout.write(
      `\n  ${B}Pass rate ${Z} ${bar(passRate)}  ${pctColor(passRate)}${B}${passRate.toFixed(1)}%${Z}\n`
    )

    try {
      const summaryPath = join(process.cwd(), 'coverage', 'coverage-summary.json')
      const raw = readFileSync(summaryPath, 'utf-8')
      const summary = JSON.parse(raw)
      const t = summary.total
      if (t) {
        const metrics = [
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
            `  ${D}(${covered}/${tot})${Z}\n`
          )
        }
      }
    } catch {
      // coverage-summary.json chưa có — bỏ qua
    }

    process.stdout.write('\n')
  }
}

module.exports = TestCaseReporter
