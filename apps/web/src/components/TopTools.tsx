import { useEffect, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { getPaperAccount, listAnalysisProfiles, resetPaperAccount } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { AnalysisProfile, PaperAccount } from '@ai-stock/types'
import { PROFILE_KEY } from '../lib/cache'
import { notifyPaperUpdated, PAPER_EVENT } from '../lib/paper'

export default function TopTools() {
  return (
    <div className="top-tools">
      <ToolFlyout
        label="模拟"
        icon={
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden>
            <path
              fill="currentColor"
              d="M3 5h18v2H3V5zm0 6h12v2H3v-2zm0 6h18v2H3v-2zm14-5.5 5 3.5-5 3.5v-7z"
            />
          </svg>
        }
      >
        <PaperFlyout />
      </ToolFlyout>
      <ToolFlyout
        label="设置"
        icon={
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden>
            <path
              fill="currentColor"
              d="M19.14 12.94c.04-.31.06-.63.06-.94s-.02-.63-.06-.94l2.03-1.58a.5.5 0 0 0 .12-.64l-1.92-3.32a.5.5 0 0 0-.6-.22l-2.39.96a7.1 7.1 0 0 0-1.63-.94l-.36-2.54a.5.5 0 0 0-.5-.42h-3.84a.5.5 0 0 0-.5.42l-.36 2.54c-.58.23-1.12.54-1.63.94l-2.39-.96a.5.5 0 0 0-.6.22L2.7 8.84a.5.5 0 0 0 .12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94L2.82 14.5a.5.5 0 0 0-.12.64l1.92 3.32c.14.24.43.34.68.22l2.39-.96c.5.4 1.05.72 1.63.94l.36 2.54c.05.24.26.42.5.42h3.84c.24 0 .45-.18.5-.42l.36-2.54c.58-.22 1.12-.54 1.63-.94l2.39.96c.25.12.54.02.68-.22l1.92-3.32a.5.5 0 0 0-.12-.64l-2.03-1.58zM12 15.5A3.5 3.5 0 1 1 12 8.5a3.5 3.5 0 0 1 0 7z"
            />
          </svg>
        }
      >
        <SettingsFlyout />
      </ToolFlyout>
    </div>
  )
}

function ToolFlyout({
  label,
  icon,
  children,
}: {
  label: string
  icon: ReactNode
  children: ReactNode
}) {
  return (
    <div className="tool-flyout">
      <button type="button" className="tool-icon" aria-label={label} title={label}>
        {icon}
      </button>
      <div className="tool-panel panel">{children}</div>
    </div>
  )
}

function PaperFlyout() {
  const [acc, setAcc] = useState<PaperAccount | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    const load = () => {
      getPaperAccount()
        .then(setAcc)
        .catch(() => setAcc(null))
    }
    load()
    window.addEventListener(PAPER_EVENT, load)
    return () => window.removeEventListener(PAPER_EVENT, load)
  }, [])

  async function reset() {
    if (!window.confirm('清空持仓和成交，资金回到 100 万？')) return
    setBusy(true)
    try {
      const next = await resetPaperAccount()
      setAcc(next)
      notifyPaperUpdated()
    } finally {
      setBusy(false)
    }
  }

  if (!acc) return <p className="muted">加载模拟账户…</p>

  return (
    <div className="flyout-body">
      <div className="flyout-title">
        <strong>模拟账户</strong>
        <Link to="/paper">完整页</Link>
      </div>
      <div className="flyout-metrics">
        <div>
          <span className="muted">总权益</span>
          <strong>{acc.equity.toFixed(0)}</strong>
        </div>
        <div>
          <span className="muted">现金</span>
          <strong>{acc.cash.toFixed(0)}</strong>
        </div>
        <div>
          <span className="muted">盈亏</span>
          <strong className={changeTone(acc.pnlPct)}>{formatChange(acc.pnlPct)}</strong>
        </div>
      </div>
      {acc.positions.length === 0 ? (
        <p className="muted tight">暂无持仓。在对话里分析股票后点「模拟」。</p>
      ) : (
        <ul className="flyout-list">
          {acc.positions.slice(0, 5).map((p) => (
            <li key={p.symbol}>
              <Link to={`/stock/${p.symbol}`}>
                {p.name} <span className="muted">{p.symbol}</span>
              </Link>
              <span className={changeTone(p.pnl)}>
                {p.qty}股 · {p.pnl >= 0 ? '+' : ''}
                {p.pnl.toFixed(0)}
              </span>
            </li>
          ))}
        </ul>
      )}
      <button type="button" className="ghost-btn" disabled={busy} onClick={() => void reset()}>
        {busy ? '重置中…' : '重置账户'}
      </button>
    </div>
  )
}

function SettingsFlyout() {
  const [profiles, setProfiles] = useState<AnalysisProfile[]>([])
  const [current, setCurrent] = useState(localStorage.getItem(PROFILE_KEY) || 'stock_analysis_default')

  useEffect(() => {
    listAnalysisProfiles()
      .then(setProfiles)
      .catch(() => setProfiles([]))
  }, [])

  function choose(code: string) {
    setCurrent(code)
    localStorage.setItem(PROFILE_KEY, code)
  }

  return (
    <div className="flyout-body">
      <div className="flyout-title">
        <strong>分析设置</strong>
        <Link to="/settings">完整页</Link>
      </div>
      <p className="muted tight">密钥只读环境变量。未配置时走量化规则兜底。</p>
      <p className="muted tight">
        <Link to="/settings">查看每日推荐 / 选股记录 →</Link>
      </p>
      <div className="flyout-profiles">
        {profiles.map((p) => (
          <button
            key={p.code}
            type="button"
            className={`pill ${current === p.code ? 'on' : ''}`}
            onClick={() => choose(p.code)}
          >
            {p.name}
            {!p.ready ? ' · 未就绪' : ''}
          </button>
        ))}
        {profiles.length === 0 && <span className="muted">加载中或服务未启动</span>}
      </div>
    </div>
  )
}
