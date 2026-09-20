import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getIndexQuotes, searchStocks } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { Quote, Stock } from '@ai-stock/types'

export default function Home() {
  const [q, setQ] = useState('')
  const [list, setList] = useState<Stock[]>([])
  const [indices, setIndices] = useState<Quote[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    getIndexQuotes()
      .then(setIndices)
      .catch(() => setIndices([]))
  }, [])

  async function onSearch(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      setList(await searchStocks(q))
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
    </main>
  )
}
