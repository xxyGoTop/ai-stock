import { useEffect, useMemo, useState } from 'react'
import { clearWatchlist, removeWatchItem } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { WatchItem } from '@ai-stock/types'
import { notifyWatchUpdated } from '../lib/watch'

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
  const [hidden, setHidden] = useState<Set<string>>(new Set())
  const [cleared, setCleared] = useState(false)
  const [busy, setBusy] = useState('')
  const sig = items.map((it) => it.symbol).join(',')

  useEffect(() => {
    setHidden(new Set())
    setCleared(false)
  }, [sig])

  const visible = useMemo(() => {
    if (cleared) return []
    return items.filter((it) => !hidden.has(it.symbol))
  }, [items, hidden, cleared])

  const filtered = useMemo(() => {
    if (mode !== 'watchlist' || tab === 'all') return visible
    if (tab === 'tomorrow_plan') {
      return visible.filter((it) => (it.category || 'default') === 'tomorrow_plan')
    }
    return visible.filter((it) => (it.category || 'default') !== 'tomorrow_plan')
  }, [visible, tab, mode])

  const total = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, total - 1)
  const slice = filtered.slice(safePage * pageSize, safePage * pageSize + pageSize)

  const defCount = visible.filter((it) => (it.category || 'default') !== 'tomorrow_plan').length
  const planCount = visible.filter((it) => (it.category || 'default') === 'tomorrow_plan').length
  const showPlan = mode !== 'watchlist' || tab === 'tomorrow_plan' || tab === 'all'

  async function removeOne(symbol: string) {
    setBusy(symbol)
    try {
      await removeWatchItem(symbol)
      setHidden((prev) => new Set([...prev, symbol]))
      notifyWatchUpdated()
    } catch (err) {
      window.alert(err instanceof Error ? err.message : '删除失败')
    } finally {
      setBusy('')
    }
  }

  async function clearAll() {
    if (!window.confirm(`清空全部自选（${visible.length} 只）？此操作不可恢复。`)) return
    setBusy('clear')
    try {
      await clearWatchlist()
      setCleared(true)
      notifyWatchUpdated()
    } catch (err) {
      window.alert(err instanceof Error ? err.message : '清空失败')
    } finally {
      setBusy('')
    }
  }

  if (visible.length === 0) {
    return <p className="muted">{cleared || hidden.size ? '自选已清空。' : '暂无标的'}</p>
  }

  return (
    <div className="watch-plan-list">
      {mode === 'watchlist' && (
        <div className="watch-tabs">
          {(
            [
              { id: 'all' as const, label: `全部 ${visible.length}` },
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
          {visible.length > 0 && (
            <button type="button" className="pill danger" disabled={busy === 'clear'} onClick={() => void clearAll()}>
              {busy === 'clear' ? '清空中…' : '一键清空'}
            </button>
          )}
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
              <button
                type="button"
                className="pill danger"
                disabled={busy === it.symbol}
                onClick={() => void removeOne(it.symbol)}
              >
                {busy === it.symbol ? '删除中…' : '删除'}
              </button>
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
