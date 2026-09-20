import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getHotFeed, getIndexQuotes, searchStocks } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { HotFeed, Quote, Stock } from '@ai-stock/types'
import { loadHot, loadScreening, loadSearch, saveHot, saveSearch } from '../lib/cache'

export default function Home() {
  const search = loadSearch()
  const [q, setQ] = useState(search.q)
  const [list, setList] = useState<Stock[]>(search.list)
  const [indices, setIndices] = useState<Quote[]>([])
  const [feed, setFeed] = useState<HotFeed | null>(loadHot()?.feed || null)
  const [error, setError] = useState('')
  const picks = loadScreening()?.result?.picks.slice(0, 6) || []

  useEffect(() => {
    getIndexQuotes()
      .then(setIndices)
      .catch(() => setIndices([]))
    const cached = loadHot()
    if (cached && Date.now() - cached.at < 60_000) return
    getHotFeed(8)
      .then((data) => {
        setFeed(data)
        saveHot(data)
      })
      .catch(() => undefined)
  }, [])

  async function onSearch(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const items = await searchStocks(q)
      setList(items)
      saveSearch(q, items)
    } catch (err) {
      setError(err instanceof Error ? err.message : '搜索失败')
    }
  }

  return (
    <main>
      <section className="panel">
        <form className="search-box" onSubmit={onSearch}>
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="输入股票名称或代码，例如 贵州茅台 / 600519"
          />
          <button type="submit">搜索</button>
        </form>
        {error && <p className="warn">{error}</p>}
        {list.length > 0 && (
          <div className="results">
            {list.map((s) => (
              <Link className="row" key={`${s.market}${s.symbol}`} to={`/stock/${s.symbol}`}>
                <strong>
                  {s.name} <span className="muted">{s.symbol}</span>
                </strong>
                <span className="muted">{s.market}</span>
              </Link>
            ))}
          </div>
        )}
      </section>

      <div className="indices">
        {indices.map((idx) => (
          <article className="panel idx" key={idx.symbol}>
            <div className="name">{idx.name}</div>
            <div className={`price ${changeTone(idx.changePercent)}`}>{idx.price.toFixed(2)}</div>
            <div className={changeTone(idx.changePercent)}>{formatChange(idx.changePercent)}</div>
          </article>
        ))}
      </div>

      {feed && (feed.topics.length > 0 || feed.boards.length > 0) && (
        <section className="panel">
          <div className="header-line">
            <h2 className="section-title">今日热点</h2>
            <Link className="muted" to="/hot">
              全部 →
            </Link>
          </div>
          <div className="algo-pills">
            {feed.topics.slice(0, 8).map((t) => (
              <span className="pill on" key={t.topic}>
                {t.topic}
              </span>
            ))}
          </div>
          <div className="board-grid compact">
            {feed.boards.slice(0, 6).map((b) => (
              <div className="card mini" key={b.code + b.name}>
                <strong>{b.name}</strong>
                <span className={changeTone(b.changePercent)}>{formatChange(b.changePercent)}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      {picks.length > 0 && (
        <section className="panel" style={{ marginTop: 16 }}>
          <div className="header-line">
            <h2 className="section-title">上次选股</h2>
            <Link className="muted" to="/screening">
              查看全部 →
            </Link>
          </div>
          <div className="results">
            {picks.map((p) => (
              <Link className="row" key={p.symbol} to={`/stock/${p.symbol}`}>
                <div>
                  <strong>
                    {p.name} <span className="muted">{p.symbol}</span>
                  </strong>
                  <div className="muted">{p.strategies.map((s) => s.short).join(' · ')}</div>
                </div>
                <span className={changeTone(p.changePercent)}>{formatChange(p.changePercent)}</span>
              </Link>
            ))}
          </div>
        </section>
      )}
    </main>
  )
}
