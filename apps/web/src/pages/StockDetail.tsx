import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getIndicators, getKline, getStock } from '@ai-stock/api-client'
import AnalysisPanel from '../components/AnalysisPanel'
import DailyNote from '../components/DailyNote'
import KlineChart from '../components/KlineChart'
import PaperTicket from '../components/PaperTicket'
import { changeTone, formatChange } from '@ai-stock/business'
import type { DataFreshness, IndicatorPoint, KlineBar, Quote } from '@ai-stock/types'

export default function StockDetail() {
  const { symbol = '' } = useParams()
  const [quote, setQuote] = useState<Quote | null>(null)
  const [bars, setBars] = useState<KlineBar[]>([])
  const [series, setSeries] = useState<IndicatorPoint[]>([])
  const [freshness, setFreshness] = useState<DataFreshness | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setError('')
    setQuote(null)
    setBars([])
    setSeries([])
    setFreshness(null)
    getStock(symbol)
      .then((q) => {
        if (!cancelled) setQuote(q)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '行情加载失败')
      })
    getKline(symbol, 180)
      .then((k) => {
        if (cancelled) return
        setBars(k.bars)
        setFreshness(k.freshness)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'K线加载失败')
      })
    getIndicators(symbol, 180)
      .then((ind) => {
        if (cancelled) return
        setSeries(ind.series)
        if (ind.freshness) setFreshness(ind.freshness)
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [symbol])

  const latest = series[series.length - 1]
  const tone = quote ? changeTone(quote.changePercent) : 'flat'

  return (
    <main className="detail-page">
      <div className="crumb">
        <Link className="back" to="/">
          行情
        </Link>
        <span>/</span>
        <Link className="back" to="/screening">
          选股
        </Link>
        <span>/</span>
        <span>{quote?.name || symbol}</span>
      </div>
      {error && <p className="warn">{error}</p>}
      <DailyNote symbol={symbol} />
      <section className="panel quote-panel">
        {quote ? (
          <>
            <div className="header-line">
              <div>
                <h1 className="quote-name">
                  {quote.name} <span className="muted">{quote.symbol}</span>
                </h1>
                <div className="muted">{quote.industry || quote.market}</div>
              </div>
              <div className="quote-price">
                <div className={`price ${tone}`}>{quote.price.toFixed(2)}</div>
                <div className={tone}>
                  {quote.change.toFixed(2)} {formatChange(quote.changePercent)}
                </div>
              </div>
            </div>
            {freshness?.stale && <p className="warn">{freshness.message}</p>}
            {!freshness?.stale && freshness?.message && <p className="muted tight">{freshness.message}</p>}
            <div className="metrics">
              <div className="metric">
                <span>开盘 / 昨收</span>
                {quote.open.toFixed(2)} / {quote.prevClose.toFixed(2)}
              </div>
              <div className="metric">
                <span>最高 / 最低</span>
                {quote.high.toFixed(2)} / {quote.low.toFixed(2)}
              </div>
              <div className="metric">
                <span>量比 / 换手</span>
                {quote.volumeRatio.toFixed(2)} / {quote.turnover.toFixed(2)}%
              </div>
              <div className="metric">
                <span>MACD / RSI6 / 乖离</span>
                {fmt(latest?.hist)} / {fmt(latest?.rsi6)} / {fmt(latest?.bias5)}%
              </div>
            </div>
          </>
        ) : (
          <p className="muted">行情加载中…</p>
        )}
        <KlineChart bars={bars} series={series} />
      </section>
      {quote && <PaperTicket symbol={symbol} name={quote.name} price={quote.price} />}
      {quote && <AnalysisPanel symbol={symbol} />}
    </main>
  )
}

function fmt(v: number | null | undefined) {
  return v == null ? '-' : v.toFixed(2)
}
