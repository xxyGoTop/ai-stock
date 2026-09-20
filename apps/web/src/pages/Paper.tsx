import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getPaperAccount } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { PaperAccount } from '@ai-stock/types'

export default function Paper() {
  const [acc, setAcc] = useState<PaperAccount | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    getPaperAccount()
      .then(setAcc)
      .catch((err) => setError(err instanceof Error ? err.message : '账户加载失败'))
  }, [])

  if (error) return <p className="warn">{error}</p>
  if (!acc) return <p className="muted">加载模拟账户…</p>

  return (
    <main>
      <section className="panel">
        <h2 className="section-title">模拟账户</h2>
        <p className="muted tight">初始 100 万，限价成交，买入当日 T+1 可卖。手续费万 2.5，卖出另加印花税 0.1%。</p>
        <div className="metrics">
          <div className="metric">
            <span>总资产</span>
            {acc.equity.toFixed(2)}
          </div>
          <div className="metric">
            <span>可用资金</span>
            {acc.cash.toFixed(2)}
          </div>
          <div className="metric">
            <span>持仓市值</span>
            {acc.marketValue.toFixed(2)}
          </div>
          <div className="metric">
            <span>累计盈亏</span>
            <b className={changeTone(acc.pnl)}>{acc.pnl.toFixed(2)} {formatChange(acc.pnlPct)}</b>
          </div>
        </div>
      </section>
      <section className="panel" style={{ marginTop: 16 }}>
        <h3 className="section-title">持仓</h3>
        {acc.positions.length === 0 && <p className="muted">暂无持仓，去个股详情页模拟买入。</p>}
        <div className="results">
          {acc.positions.map((p) => (
            <Link className="row" key={p.symbol} to={`/stock/${p.symbol}`}>
              <div>
                <strong>
                  {p.name} <span className="muted">{p.symbol}</span>
                </strong>
                <div className="muted">
                  {p.qty} 股 · 可卖 {p.available} · 成本 {p.cost.toFixed(2)}
                </div>
              </div>
              <div className="pick-side">
                <div>{p.price.toFixed(2)}</div>
                <div className={changeTone(p.pnl)}>{p.pnl.toFixed(2)}</div>
              </div>
            </Link>
          ))}
        </div>
      </section>
      <section className="panel" style={{ marginTop: 16 }}>
        <h3 className="section-title">成交</h3>
        <div className="results">
          {acc.orders.map((o) => (
            <div className="row" key={o.id + o.createdAt}>
              <div>
                <strong>
                  {o.side === 'buy' ? '买' : '卖'} {o.name} <span className="muted">{o.symbol}</span>
                </strong>
                <div className="muted">
                  {o.createdAt} · {o.qty} 股 · 费 {o.fee.toFixed(2)}
                </div>
              </div>
              <div>{o.price.toFixed(2)}</div>
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}
