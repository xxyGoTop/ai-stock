import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getHotFeed, getIndexQuotes, getWatchlist, searchStocks } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { HotFeed, Quote, Stock, WatchItem } from '@ai-stock/types'
import WatchButton from '../components/WatchButton'
import { loadHot, loadScreening, loadSearch, saveHot, saveSearch } from '../lib/cache'
import { WATCH_EVENT } from '../lib/watch'

export default function Home() {
  const search = loadSearch()
  const [q, setQ] = useState(search.q)
  const [list, setList] = useState<Stock[]>(search.list)
  const [indices, setIndices] = useState<Quote[]>([])
  const [feed, setFeed] = useState<HotFeed | null>(loadHot()?.feed || null)
  const [watch, setWatch] = useState<WatchItem[]>([])
  const [error, setError] = useState('')
  const picks = loadScreening()?.result?.picks.slice(0, 6) || []

  useEffect(() => {
    const loadWatch = () => {
      getWatchlist()
        .then((data) => setWatch(data.items))
        .catch(() => setWatch([]))
    }
    loadWatch()
    window.addEventListener(WATCH_EVENT, loadWatch)
    getIndexQuotes()
      .then(setIndices)
      .catch(() => setIndices([]))
    const cached = loadHot()
    if (!cached || Date.now() - cached.at >= 60_000) {
      getHotFeed(8)
        .then((data) => {
          setFeed(data)
          saveHot(data)
        })
        .catch(() => undefined)
    }
    return () => window.removeEventListener(WATCH_EVENT, loadWatch)
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
              <div className="row" key={`${s.market}${s.symbol}`}>
                <Link className="pick-main" to={`/stock/${s.symbol}`}>
                  <strong>
                    {s.name} <span className="muted">{s.symbol}</span>
                  </strong>
                </Link>
                <WatchButton symbol={s.symbol} name={s.name} compact />
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="panel" style={{ marginTop: 16 }}>
        <div className="header-line">
          <h2 className="section-title">自选股</h2>
          <span className="muted">{watch.length ? `${watch.length} 只` : '还没有自选'}</span>
        </div>
        {watch.length === 0 ? (
          <p className="muted">搜索股票或打开个股详情，点「加入自选」后会显示在这里。</p>
        ) : (
          <div className="results">
            {watch.map((it) => (
              <div className="row" key={it.symbol}>
                <Link className="pick-main" to={`/stock/${it.symbol}`}>
                  <strong>
                    {it.name} <span className="muted">{it.symbol}</span>
                  </strong>
                  <div className="muted">{it.industry || it.market}</div>
                </Link>
                <div className="pick-side">
                  <div className={changeTone(it.changePercent || 0)}>{it.price ? it.price.toFixed(2) : '-'}</div>
                  <div className={changeTone(it.changePercent || 0)}>{formatChange(it.changePercent || 0)}</div>
                  <WatchButton symbol={it.symbol} name={it.name} compact />
                </div>
              </div>
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
