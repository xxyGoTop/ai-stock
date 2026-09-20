import { useEffect, useRef } from 'react'
import * as echarts from 'echarts'
import type { IndicatorPoint, KlineBar } from '@ai-stock/types'

export default function KlineChart({ bars, series }: { bars: KlineBar[]; series: IndicatorPoint[] }) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el || bars.length === 0) return
    const chart = echarts.init(el)
    const dates = bars.map((b) => b.date)
    const ma = (key: keyof IndicatorPoint) =>
      bars.map((b) => {
        const hit = series.find((p) => p.date === b.date)
        const v = hit?.[key]
        return typeof v === 'number' ? v : null
      })
    chart.setOption({
      backgroundColor: 'transparent',
      animation: false,
      legend: { data: ['K线', 'MA5', 'MA10', 'MA20'], textStyle: { color: '#9aa8c2' }, top: 4 },
      tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
      axisPointer: { link: [{ xAxisIndex: 'all' }] },
      grid: [
        { left: 56, right: 18, top: 36, height: '58%' },
        { left: 56, right: 18, top: '76%', height: '16%' },
      ],
      xAxis: [
        { type: 'category', data: dates, axisLine: { lineStyle: { color: '#24324a' } }, axisLabel: { color: '#8b97ad' } },
        { type: 'category', data: dates, gridIndex: 1, axisLine: { lineStyle: { color: '#24324a' } }, axisLabel: { show: false } },
      ],
      yAxis: [
        { scale: true, splitLine: { lineStyle: { color: '#1a2438' } }, axisLabel: { color: '#8b97ad' } },
        { gridIndex: 1, splitLine: { show: false }, axisLabel: { color: '#8b97ad' } },
      ],
      dataZoom: [
        { type: 'inside', xAxisIndex: [0, 1], start: 55 },
        { type: 'slider', xAxisIndex: [0, 1], start: 55, height: 18, bottom: 6, borderColor: '#24324a', textStyle: { color: '#8b97ad' } },
      ],
      series: [
        {
          name: 'K线',
          type: 'candlestick',
          data: bars.map((b) => [b.open, b.close, b.low, b.high]),
          itemStyle: { color: '#ef4444', color0: '#22c55e', borderColor: '#ef4444', borderColor0: '#22c55e' },
        },
        { name: 'MA5', type: 'line', showSymbol: false, data: ma('ma5'), lineStyle: { width: 1.3, color: '#f5c16c' } },
        { name: 'MA10', type: 'line', showSymbol: false, data: ma('ma10'), lineStyle: { width: 1.2, color: '#60a5fa' } },
        { name: 'MA20', type: 'line', showSymbol: false, data: ma('ma20'), lineStyle: { width: 1.2, color: '#c084fc' } },
        {
          name: '成交量',
          type: 'bar',
          xAxisIndex: 1,
          yAxisIndex: 1,
          data: bars.map((b) => ({
            value: b.volume,
            itemStyle: { color: b.close >= b.open ? '#7f1d1d' : '#14532d' },
          })),
        },
      ],
    })
    const tick = () => chart.resize()
    requestAnimationFrame(tick)
    window.addEventListener('resize', tick)
    return () => {
      window.removeEventListener('resize', tick)
      chart.dispose()
    }
  }, [bars, series])

  if (bars.length === 0) {
    return <div className="chart chart-empty">K 线加载中…</div>
  }
  return <div ref={ref} className="chart" />
}
