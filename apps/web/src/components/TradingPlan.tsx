import { useState } from 'react'
import { Link } from 'react-router-dom'
import { placePaperOrder } from '@ai-stock/api-client'
import type { DailyNote } from '@ai-stock/types'
import { notifyPaperUpdated } from '../lib/paper'

const RISK: Record<string, string> = { low: '偏低', medium: '中等', high: '偏高' }

export default function TradingPlan({ note }: { note: DailyNote }) {
  const p = note.plan
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')
  const qty = p.suggestedQty && p.suggestedQty >= 100 ? p.suggestedQty : 100
  const live = note.price || 0
  const chasing = live > 0 && live > p.buyHigh
  const below = live > 0 && live < p.buyLow
  const limit = chasing ? live : below ? p.buyLow : live || p.buyHigh
  const label = chasing ? '按现价模拟买入' : below ? '按区间下限挂单' : '按现价模拟买入'

  async function confirm() {
    setBusy(true)
    setMsg('')
    try {
      await placePaperOrder({ symbol: note.symbol, name: note.name, side: 'buy', price: limit, qty })
      notifyPaperUpdated()
      setOpen(false)
      setMsg(`已模拟买入 ${qty} 股，成交价 ${limit.toFixed(2)}`)
    } catch (err) {
      setMsg(err instanceof Error ? err.message : '下单失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="trade-plan">
      <div className="sv-h">交易计划</div>
      <div className="plan-meta">
        <div>
          <span>关注区间</span>
          <b>
            {p.buyLow.toFixed(2)} - {p.buyHigh.toFixed(2)}
          </b>
        </div>
        <div>
          <span>模拟仓位</span>
          <b>{Math.round((p.position || 0.1) * 100)}%</b>
          <em>{qty} 股</em>
        </div>
        <div>
          <span>风险</span>
          <b>{RISK[p.riskLevel || 'medium'] || p.riskLevel}</b>
        </div>
      </div>
      {p.invalidConditions?.length ? (
        <div className="invalid">
          <span>失效条件</span>
          <ul>
            {p.invalidConditions.map((x) => (
              <li key={x}>{x}</li>
            ))}
          </ul>
        </div>
      ) : null}
      <p className="disclaimer">AI 交易计划仅用于投资研究与模拟交易，不构成投资建议。</p>
      <div className="ticket-row">
        <button type="button" className="ghost buy-btn" onClick={() => setOpen(true)}>
          {label}
        </button>
        <a className="ghost-btn" href="#paper-ticket">
          去下方改价下单
        </a>
        <Link className="ghost-btn" to="/paper">
          查看模拟账户
        </Link>
      </div>
      {chasing ? (
        <p className="warn tight">现价 {live.toFixed(2)} 已高出买入上限 {p.buyHigh.toFixed(2)}，按钮按现价成交，不追区间挂单。</p>
      ) : null}
      {below ? (
        <p className="muted tight">现价尚未进入区间，确认后按下限 {p.buyLow.toFixed(2)} 挂模拟单。</p>
      ) : null}
      {msg ? <p className="muted">{msg}</p> : null}
      {open ? (
        <div className="confirm-mask" onClick={() => !busy && setOpen(false)}>
          <div className="confirm-box" onClick={(e) => e.stopPropagation()}>
            <h3>确认模拟买入 {note.name}</h3>
            <p>
              现价 {live.toFixed(2)} · 数量 {qty} 股 · 成交 {limit.toFixed(2)}
            </p>
            <p>
              计划区间 {p.buyLow.toFixed(2)} ~ {p.buyHigh.toFixed(2)} · 止损 {p.stop.toFixed(2)} · 目标 {p.target1.toFixed(2)} /{' '}
              {p.target2.toFixed(2)}
            </p>
            {chasing ? <p className="warn">这是追高按现价模拟，不是计划里的回踩买入。</p> : null}
            <p className="muted">仅本地模拟账户，T+1，不接入真实券商。</p>
            <div className="ticket-row">
              <button type="button" className="ghost-btn" disabled={busy} onClick={() => setOpen(false)}>
                取消
              </button>
              <button type="button" className="ghost buy-btn" disabled={busy} onClick={() => void confirm()}>
                {busy ? '下单中…' : '确认下单'}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
