import { useEffect, useState } from 'react'
import { getPaperAccount, placePaperOrder } from '@ai-stock/api-client'
import type { PaperAccount, PaperPosition } from '@ai-stock/types'
import { notifyPaperUpdated, PAPER_EVENT } from '../lib/paper'

export default function PaperTicket({ symbol, name, price }: { symbol: string; name: string; price: number }) {
  const [qty, setQty] = useState(100)
  const [limit, setLimit] = useState(price)
  const [acc, setAcc] = useState<PaperAccount | null>(null)
  const [msg, setMsg] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    setLimit(price)
  }, [price, symbol])

  useEffect(() => {
    const load = () => {
      getPaperAccount()
        .then(setAcc)
        .catch(() => setAcc(null))
    }
    load()
    window.addEventListener(PAPER_EVENT, load)
    return () => window.removeEventListener(PAPER_EVENT, load)
  }, [symbol])

  const pos: PaperPosition | undefined = acc?.positions.find((p) => p.symbol === symbol)

  async function submit(side: 'buy' | 'sell') {
    setBusy(true)
    setMsg('')
    try {
      const next = await placePaperOrder({ symbol, name, side, price: limit || price, qty })
      setAcc(next)
      notifyPaperUpdated()
      setMsg(side === 'buy' ? '模拟买入已成交' : '模拟卖出已成交')
    } catch (err) {
      setMsg(err instanceof Error ? err.message : '下单失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="panel paper-ticket">
      <div className="header-line wrap">
        <h2 className="section-title">模拟下单</h2>
        <span className="muted tiny">仅本地模拟账户，T+1，非实盘</span>
      </div>
      {acc ? (
        <p className="muted tight">
          可用资金 {acc.cash.toFixed(2)}
          {pos ? ` · 持仓 ${pos.qty} 股，可卖 ${pos.available}，成本 ${pos.cost.toFixed(2)}` : ' · 当前无此票持仓'}
        </p>
      ) : null}
      <div className="ticket-row">
        <label>
          限价
          <input type="number" step="0.01" value={limit || price} onChange={(e) => setLimit(Number(e.target.value))} />
        </label>
        <label>
          数量
          <input type="number" step="100" min="100" value={qty} onChange={(e) => setQty(Number(e.target.value))} />
        </label>
        <button className="ghost buy-btn" disabled={busy} onClick={() => submit('buy')}>
          买入
        </button>
        <button className="ghost sell-btn" disabled={busy} onClick={() => submit('sell')}>
          卖出
        </button>
      </div>
      {msg && <p className="muted">{msg}</p>}
    </section>
  )
}
