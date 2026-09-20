export type Market = 'SH' | 'SZ'

export interface Stock {
  symbol: string
  name: string
  market: Market
}

export interface Quote extends Stock {
  price: number
  change: number
  changePercent: number
  open: number
  high: number
  low: number
  prevClose: number
  volume: number
  amount: number
  turnover: number
  volumeRatio: number
  amplitude: number
  industry: string
}

export interface KlineBar {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  amount: number
  changePercent: number
  turnover: number
  intraday?: boolean
}

export interface DataFreshness {
  klineDate: string
  quoteDate: string
  stale: boolean
  patchedIntraday: boolean
  message: string
}

export interface IndicatorPoint {
  date: string
  ma5: number | null
  ma10: number | null
  ma20: number | null
  dif: number | null
  dea: number | null
  hist: number | null
  rsi6: number | null
  k: number | null
  d: number | null
  j: number | null
  bias5: number | null
}

export interface IndicatorResult {
  symbol: string
  freshness: DataFreshness
  latest: IndicatorPoint | null
  series: IndicatorPoint[]
}

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
