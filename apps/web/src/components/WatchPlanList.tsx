import { useMemo, useState } from 'react'
import { changeTone, formatChange } from '@ai-stock/business'
import type { WatchItem } from '@ai-stock/types'

type Props = {
  items: WatchItem[]
  pageSize?: number
  mode?: 'watchlist' | 'tomorrow_plan' | 'today_ops'
  onPick: (symbol: string, name?: string) => void
  onAction: (action: string, symbol?: string, label?: string) => void
}

function planText(it: WatchItem): string {
  const p = it.plan
  if (!p) return '计划待补充'
  const parts: string[] = []
  if (p.action) parts.push(p.action)
  if ((p.buyLow || 0) > 0 || (p.buyHigh || 0) > 0) {
    parts.push(`站稳 ${(p.buyLow || 0).toFixed(2)}-${(p.buyHigh || 0).toFixed(2)} 买`)
  }
  if ((p.stop || 0) > 0) parts.push(`止损 ${p.stop!.toFixed(2)}`)
  if ((p.target1 || 0) > 0) {
    let t = `目标 ${p.target1!.toFixed(2)}`
    if ((p.target2 || 0) > 0) t += `/${p.target2!.toFixed(2)}`
    parts.push(t)
  }
  if ((p.buyBatches || 0) > 0) parts.push(`分 ${p.buyBatches} 次买`)
  if ((p.position || 0) > 0) parts.push(`仓位 ${Math.round((p.position || 0) * 100)}%`)
  return parts.length ? parts.join(' · ') : '计划待补充'
}

export default function WatchPlanList({ items, pageSize = 8, mode = 'watchlist', onPick, onAction }: Props) {
  const [tab, setTab] = useState<'all' | 'default' | 'tomorrow_plan'>(
    mode === 'tomorrow_plan' || mode === 'today_ops' ? 'tomorrow_plan' : 'all',
  )
  const [page, setPage] = useState(0)

  const filtered = useMemo(() => {
    if (mode !== 'watchlist' || tab === 'all') return items
    if (tab === 'tomorrow_plan') {
      return items.filter((it) => (it.category || 'default') === 'tomorrow_plan')
    }
    return items.filter((it) => (it.category || 'default') !== 'tomorrow_plan')
  }, [items, tab, mode])

  const total = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, total - 1)
  const slice = filtered.slice(safePage * pageSize, safePage * pageSize + pageSize)

  const defCount = items.filter((it) => (it.category || 'default') !== 'tomorrow_plan').length
  const planCount = items.filter((it) => (it.category || 'default') === 'tomorrow_plan').length
  const showPlan = mode !== 'watchlist' || tab === 'tomorrow_plan' || tab === 'all'

  if (items.length === 0) {
    return <p className="muted">暂无标的</p>
  }

  return (
    <div className="watch-plan-list">
      {mode === 'watchlist' && (
        <div className="watch-tabs">
          {(
            [
              { id: 'all' as const, label: `全部 ${items.length}` },
              { id: 'default' as const, label: `普通 ${defCount}` },
              { id: 'tomorrow_plan' as const, label: `明日计划 ${planCount}` },
            ] as const
          ).map((t) => (
            <button
              key={t.id}
              type="button"
              className={`pill ${tab === t.id ? 'on' : ''}`}
              onClick={() => {
                setTab(t.id)
                setPage(0)
              }}
            >
              {t.label}
            </button>
          ))}
        </div>
      )}

      <div className="pick-list">
        {slice.map((it, idx) => (
          <div className="pick-row watch-plan-row" key={`${it.symbol}-${safePage}-${idx}`}>
            <div className="watch-plan-main">
              <button type="button" className="linkish" onClick={() => onPick(it.symbol, it.name)}>
                <strong>
                  <span className="muted">{safePage * pageSize + idx + 1}. </span>
                  {it.name} <span className="muted">{it.symbol}</span>
                </strong>
                {typeof it.changePercent === 'number' && (
                  <span className={changeTone(it.changePercent)}>{formatChange(it.changePercent)}</span>
                )}
              </button>
              {(it.category || 'default') === 'tomorrow_plan' && (
                <span className="plan-tag">{mode === 'today_ops' ? '今日操作' : '明日计划'}</span>
              )}
              {it.planForDate && <span className="muted plan-date">计划日 {it.planForDate}</span>}
              {showPlan && (it.plan || mode !== 'watchlist') && (
                <p className="plan-line muted">{planText(it)}</p>
              )}
              {it.plan?.note && <p className="plan-note muted">{it.plan.note}</p>}
            </div>
            <div className="action-row tight">
              <button type="button" className="pill" onClick={() => onAction('analyze', it.symbol, it.name)}>
                分析
              </button>
              <button type="button" className="pill" onClick={() => onAction('kline', it.symbol, it.name)}>
                K线
              </button>
              <button type="button" className="pill" onClick={() => onAction('paper', it.symbol, it.name)}>
                模拟
              </button>
              {mode === 'watchlist' && (it.category || 'default') !== 'tomorrow_plan' && (
                <button
                  type="button"
                  className="pill"
                  onClick={() => onAction('tomorrow_plan', it.symbol, it.name)}
                >
                  明日计划
                </button>
              )}
            </div>
          </div>
        ))}
      </div>

      {filtered.length > pageSize && (
        <div className="pager">
          <button
            type="button"
            className="ghost-btn"
            disabled={safePage <= 0}
            onClick={() => setPage((p) => Math.max(0, p - 1))}
          >
            上一页
          </button>
          <span className="muted">
            {safePage + 1} / {total} · 共 {filtered.length} 只
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
