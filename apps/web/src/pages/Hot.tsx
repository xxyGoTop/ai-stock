import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getHotFeed } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { HotFeed } from '@ai-stock/types'
import { formatClock, loadHot, saveHot } from '../lib/cache'

export default function Hot() {
  const cached = loadHot()
  const [feed, setFeed] = useState<HotFeed | null>(cached?.feed || null)
  const [at, setAt] = useState(cached?.at || 0)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(!cached?.feed)

  useEffect(() => {
    const fresh = cached && Date.now() - cached.at < 60_000
    if (fresh) return
    getHotFeed(18)
      .then((data) => {
        setFeed(data)
        saveHot(data)
        setAt(Date.now())
      })
      .catch((err) => setError(err instanceof Error ? err.message : '热点加载失败'))
      .finally(() => setLoading(false))
  }, [])

  return (
    <main>
      <section className="panel">
        <div className="header-line">
          <div>
            <h2 className="section-title">今日热点</h2>
            <p className="muted tight">东财重要快讯 + 行业涨幅榜，题材由标题关键词聚类。{at ? `${formatClock(at)} 已缓存` : ''}</p>
          </div>
        </div>
        {error && <p className="warn">{error}</p>}
        {loading && !feed && <p className="muted">加载中…</p>}
        {feed && feed.topics.length > 0 && (
          <div className="algo-pills">
            {feed.topics.map((t) => (
              <span className="pill on" key={t.topic}>
                {t.topic} · {t.heat}
              </span>
            ))}
          </div>
        )}
      </section>

      {feed && (
        <section className="split" style={{ marginTop: 16 }}>
          <div className="panel">
            <h3 className="section-title">强势行业</h3>
            <div className="board-grid">
              {feed.boards.map((b) => (
                <article className="card" key={b.code + b.name}>
                  <div className="header-line">
                    <strong>{b.name}</strong>
                    <span className={changeTone(b.changePercent)}>{formatChange(b.changePercent)}</span>
                  </div>
                  <div className="muted tiny">5日 {formatChange(b.change5 || 0)}</div>
                  {b.leader && (
                    <div className="muted">
                      领涨{' '}
                      {b.leaderCode ? (
                        <Link to={`/stock/${b.leaderCode}`}>{b.leader}</Link>
                      ) : (
                        b.leader
                      )}{' '}
                      <span className={changeTone(b.leaderChangePercent)}>{formatChange(b.leaderChangePercent)}</span>
                    </div>
                  )}
                </article>
              ))}
            </div>
          </div>
          <div className="panel">
            <h3 className="section-title">重要快讯</h3>
            <div className="news-list">
              {feed.news.map((n) => (
                <a className="news-item" key={n.code || n.title} href={n.url || '#'} target="_blank" rel="noreferrer">
                  <div className="news-meta">
                    <span>{n.time || '—'}</span>
                    <span>{n.source}</span>
                  </div>
                  <strong>{n.title}</strong>
                  {n.summary && <p className="muted tight">{n.summary}</p>}
                </a>
              ))}
            </div>
          </div>
        </section>
      )}
    </main>
  )
}
