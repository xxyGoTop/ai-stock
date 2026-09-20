import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import * as echarts from 'echarts'
import { getIndicators, getKline, getStock } from '@ai-stock/api-client'
import { changeTone, formatChange } from '@ai-stock/business'
import type { DataFreshness, IndicatorPoint, KlineBar, Quote } from '@ai-stock/types'

export default function StockDetail() {
  const { symbol = '' } = useParams()
  const chartRef = useRef<HTMLDivElement>(null)
  const [quote, setQuote] = useState<Quote | null>(null)
  const [bars, setBars] = useState<KlineBar[]>([])
  const [series, setSeries] = useState<IndicatorPoint[]>([])
  const [freshness, setFreshness] = useState<DataFreshness | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setError('')
    Promise.all([getStock(symbol), getKline(symbol, 180), getIndicators(symbol, 180)])
      .then(([q, k, ind]) => {
        if (cancelled) return
        setQuote(q)
        setBars(k.bars)
        setSeries(ind.series)
        setFreshness(ind.freshness)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      })
    return () => {
      cancelled = true
    }
  }, [symbol])

  useEffect(() => {
    if (!chartRef.current || bars.length === 0) return
    const chart = echarts.init(chartRef.current)
    const dates = bars.map((b) => b.date)
    chart.setOption({
      backgroundColor: 'transparent',
      animation: false,
      legend: { data: ['K线', 'MA5', 'MA10', 'MA20'], textStyle: { color: '#8b97ad' } },
      tooltip: { trigger: 'axis' },
      axisPointer: { link: [{ xAxisIndex: 'all' }] },
      grid: [
        { left: 48, right: 16, top: 36, height: '58%' },
        { left: 48, right: 16, top: '76%', height: '16%' },
      ],
      xAxis: [
        { type: 'category', data: dates, axisLine: { lineStyle: { color: '#1f2a40' } } },
        { type: 'category', data: dates, gridIndex: 1, axisLine: { lineStyle: { color: '#1f2a40' } } },
      ],
      yAxis: [
        { scale: true, splitLine: { lineStyle: { color: '#1f2a40' } } },
        { gridIndex: 1, splitLine: { show: false } },
      ],
      dataZoom: [{ type: 'inside', xAxisIndex: [0, 1], start: 60 }],
      series: [
        {
          name: 'K线',
          type: 'candlestick',
          data: bars.map((b) => [b.open, b.close, b.low, b.high]),
          itemStyle: { color: '#ef4444', color0: '#22c55e', borderColor: '#ef4444', borderColor0: '#22c55e' },
        },
        { name: 'MA5', type: 'line', showSymbol: false, data: series.map((p) => p.ma5), lineStyle: { width: 1.2 } },
        { name: 'MA10', type: 'line', showSymbol: false, data: series.map((p) => p.ma10), lineStyle: { width: 1.2 } },
        { name: 'MA20', type: 'line', showSymbol: false, data: series.map((p) => p.ma20), lineStyle: { width: 1.2 } },
        { name: '成交量', type: 'bar', xAxisIndex: 1, yAxisIndex: 1, data: bars.map((b) => b.volume), itemStyle: { color: '#334155' } },
      ],
    })
    const onResize = () => chart.resize()
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
      chart.dispose()
    }
  }, [bars, series])

  const latest = series[series.length - 1]
  const tone = quote ? changeTone(quote.changePercent) : 'flat'

  return (
    <main>
      <Link className="back" to="/">
        ← 返回搜索
      </Link>
      {error && <p className="warn">{error}</p>}
      {quote && (
        <section className="panel">
          <div className="header-line">
            <div>
              <h1 style={{ margin: 0 }}>
                {quote.name} <span className="muted">{quote.symbol}</span>
              </h1>
              <div className="muted">{quote.industry || quote.market}</div>
            </div>
            <div>
              <div className={`price ${tone}`} style={{ fontSize: 32 }}>
                {quote.price.toFixed(2)}
              </div>
              <div className={tone}>
                {quote.change.toFixed(2)} {formatChange(quote.changePercent)}
              </div>
            </div>
          </div>
          {freshness?.stale && <p className="warn">{freshness.message}</p>}
          {!freshness?.stale && freshness?.message && <p className="muted">{freshness.message}</p>}
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
          <div ref={chartRef} className="chart" />
        </section>
      )}
    </main>
  )
}

function fmt(v: number | null | undefined) {
  return v == null ? '-' : v.toFixed(2)
}
