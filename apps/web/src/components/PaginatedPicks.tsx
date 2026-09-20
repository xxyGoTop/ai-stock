import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { changeTone, formatChange } from '@ai-stock/business'

export type PagePick = {
  symbol: string
  name: string
  changePercent?: number
  reason?: string
  board?: string
  score?: number
  primaryName?: string
  primary?: string
  industry?: string
}

type Props = {
  items: PagePick[]
  pageSize?: number
  onPick: (symbol: string, name?: string) => void
  onAction: (action: string, symbol?: string, label?: string) => void
  showDetailLink?: boolean
}

export default function PaginatedPicks({ items, pageSize = 10, onPick, onAction, showDetailLink }: Props) {
  const [page, setPage] = useState(0)
  const total = Math.max(1, Math.ceil(items.length / pageSize))
  const safePage = Math.min(page, total - 1)
  const slice = useMemo(() => {
    const start = safePage * pageSize
    return items.slice(start, start + pageSize)
  }, [items, safePage, pageSize])

  if (items.length === 0) {
    return <p className="muted">暂无标的</p>
  }

  return (
    <div className="paged-picks">
      <div className="pick-list">
        {slice.map((p, idx) => (
          <div className="pick-row" key={`${p.symbol}-${safePage}-${idx}`}>
            <button type="button" className="linkish" onClick={() => onPick(p.symbol, p.name)}>
              <strong>
                <span className="muted">{safePage * pageSize + idx + 1}. </span>
                {p.name} <span className="muted">{p.symbol}</span>
              </strong>
              {typeof p.changePercent === 'number' && (
                <span className={changeTone(p.changePercent)}>{formatChange(p.changePercent)}</span>
              )}
              <span className="muted">
                {p.reason || p.primaryName || p.primary || ''}
                {typeof p.score === 'number' ? ` · 分 ${p.score}` : ''}
                {p.industry ? ` · ${p.industry}` : ''}
                {p.board ? ` · ${p.board}` : ''}
              </span>
            </button>
            <div className="action-row tight">
              <button type="button" className="pill" onClick={() => onAction('analyze', p.symbol, p.name)}>
                分析
              </button>
              <button type="button" className="pill" onClick={() => onAction('watch', p.symbol, p.name)}>
                自选
              </button>
              <button type="button" className="pill" onClick={() => onAction('kline', p.symbol, p.name)}>
                K线
              </button>
              <button type="button" className="pill" onClick={() => onAction('paper', p.symbol, p.name)}>
                模拟
              </button>
              {showDetailLink && (
                <Link className="pill" to={`/stock/${p.symbol}`}>
                  详情
                </Link>
              )}
            </div>
          </div>
        ))}
      </div>
      {items.length > pageSize && (
        <div className="pager">
          <button type="button" className="ghost-btn" disabled={safePage <= 0} onClick={() => setPage((p) => Math.max(0, p - 1))}>
            上一页
          </button>
          <span className="muted">
            {safePage + 1} / {total} · 共 {items.length} 只
          </span>
          <button
            type="button"
            className="ghost-btn"
            disabled={safePage >= total - 1}
            onClick={() => setPage((p) => Math.min(total - 1, p + 1))}
          >
            下一页
          </button>
        </div>
      )}
    </div>
  )
}
