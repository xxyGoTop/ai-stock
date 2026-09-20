import { useEffect, useState } from 'react'
import { placePaperOrder } from '@ai-stock/api-client'

export default function PaperTicket({ symbol, name, price }: { symbol: string; name: string; price: number }) {
  const [qty, setQty] = useState(100)
  const [limit, setLimit] = useState(price)

  useEffect(() => {
    setLimit(price)
  }, [price, symbol])
  const [msg, setMsg] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(side: 'buy' | 'sell') {
    setBusy(true)
    setMsg('')
    try {
      await placePaperOrder({ symbol, name, side, price: limit || price, qty })
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
