import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { listAlgorithms, runScreening } from '@ai-stock/api-client'
import WatchButton from '../components/WatchButton'
import { changeTone, formatChange } from '@ai-stock/business'
import type { AlgorithmMeta, ScreenResult } from '@ai-stock/types'
import { formatClock, loadScreening, saveScreening } from '../lib/cache'

export default function Screening() {
  const cached = loadScreening()
  const [algos, setAlgos] = useState<AlgorithmMeta[]>(cached?.algos || [])
  const [selected, setSelected] = useState<string[]>(cached?.selected || [])
  const [result, setResult] = useState<ScreenResult | null>(cached?.result || null)
  const [filter, setFilter] = useState(cached?.filter || 'all')
  const [at, setAt] = useState(cached?.at || 0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    listAlgorithms()
      .then((items) => {
        setAlgos(items)
        const nextSelected = loadScreening()?.selected?.length ? loadScreening()!.selected : items.map((a) => a.code)
        setSelected(nextSelected)
        saveScreening({ algos: items, selected: nextSelected })
      })
      .catch(() => {
        if (!algos.length) setAlgos([])
      })
  }, [])

  function toggle(code: string) {
    const next = selected.includes(code) ? selected.filter((x) => x !== code) : [...selected, code]
    setSelected(next)
    saveScreening({ selected: next })
  }

  async function onRun() {
    setLoading(true)
    setError('')
    try {
      const data = await runScreening({ detail: 80, limit: 30, algorithms: selected })
      const now = Date.now()
      setResult(data)
      setAt(now)
      saveScreening({ result: data, selected, filter, at: now })
    } catch (err) {
      setError(err instanceof Error ? err.message : '选股失败，请确认 Python 服务已启动')
    } finally {
      setLoading(false)
    }
  }

  function setFilterAndCache(code: string) {
    setFilter(code)
    saveScreening({ filter: code })
  }

  const picks = useMemo(() => {
    if (!result) return []
    if (filter === 'all') return result.picks
    return result.picks.filter((p) => p.strategies.some((s) => s.code === filter))
  }, [result, filter])

  return (
    <main>
      <section className="panel">
        <div className="header-line">
          <div>
            <h2 className="section-title">五算法综合选股</h2>
            <p className="muted tight">年新高 / 深调回升 / 火车轨 / 每日观察 / 五日线。扫描成交额靠前的 80 只，结果会缓存在本页会话里。</p>
          </div>
          <button className="ghost" disabled={loading || selected.length === 0} onClick={onRun}>
            {loading ? '扫描中…' : result ? '重新扫描' : '开始选股'}
          </button>
        </div>
        <div className="algo-pills">
          {algos.map((a) => (
            <label key={a.code} className={selected.includes(a.code) ? 'pill on' : 'pill'}>
              <input type="checkbox" checked={selected.includes(a.code)} onChange={() => toggle(a.code)} />
              {a.short}
            </label>
          ))}
        </div>
        {error && <p className="warn">{error}</p>}
      </section>

      {result && (
        <section className="panel" style={{ marginTop: 16 }}>
          <div className="header-line">
            <div className="muted">
              扫描 {result.scanned} · 达标 {result.qualified}
              {at ? ` · ${formatClock(at)} 已缓存` : ''}
              {algos.map((a) => ` ｜ ${a.short} ${result.strategyCount[a.code] || 0}`).join('')}
            </div>
          </div>
          <div className="algo-pills">
            <button className={filter === 'all' ? 'pill on' : 'pill'} onClick={() => setFilterAndCache('all')}>
              全部 {result.qualified}
            </button>
            {algos.map((a) => (
              <button
                key={a.code}
                className={filter === a.code ? 'pill on' : 'pill'}
                onClick={() => setFilterAndCache(a.code)}
              >
                {a.short} {result.strategyCount[a.code] || 0}
              </button>
            ))}
          </div>
          <div className="results">
            {picks.map((p) => (
              <div className="row screen-row" key={p.symbol}>
                <Link className="pick-main" to={`/stock/${p.symbol}`}>
                  <div className="pick-title">
                    <strong>{p.name}</strong>
                    <span className="muted">{p.symbol}</span>
                    {p.inFirstPage && <span className="tag-chip">涨幅榜</span>}
                    {p.industry && <span className="muted">{p.industry}</span>}
                  </div>
                  <div className="tag-row">
                    {p.strategies.map((s) => (
                      <span className="tag-chip accent" key={s.code}>
                        {s.short}
                      </span>
                    ))}
                  </div>
                  <div className="muted pick-reason">{p.strategies[0]?.reason}</div>
                </Link>
                <div className="pick-side">
                  <div className="score">{p.score.toFixed(0)}</div>
                  <div className={changeTone(p.changePercent)}>{formatChange(p.changePercent)}</div>
                  <WatchButton symbol={p.symbol} name={p.name} compact />
                </div>
              </div>
            ))}
            {picks.length === 0 && <p className="muted">当前筛选没有命中。</p>}
          </div>
        </section>
      )}
    </main>
  )
}
