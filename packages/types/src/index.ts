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

export interface AlgorithmMeta {
  code: string
  name: string
  short: string
  category: string
  baseScore: number
  enabled: boolean
}

export interface ScreenHit {
  code: string
  name: string
  short: string
  score: number
  reason: string
}

export interface ScreenPick {
  symbol: string
  name: string
  market: Market
  industry: string
  price: number
  changePercent: number
  score: number
  primary: string
  primaryName: string
  strategies: ScreenHit[]
  inFirstPage: boolean
  rps: { rps20: number; rps50: number; rps120: number; rps250: number }
}

export interface ScreenResult {
  profileCode: string
  scanned: number
  qualified: number
  strategyCount: Record<string, number>
  picks: ScreenPick[]
}

export interface AnalysisProfile {
  code: string
  name: string
  task: string
  mode: 'single' | 'fallback' | 'ensemble'
  ready: boolean
  readyModels: string[]
}

export interface LlmModel {
  code: string
  name: string
  providerCode: string
  roles: string[]
  costTier: string
  enabled: boolean
  ready: boolean
}

export interface AnalysisCard {
  cardType: string
  title: string
  score: number
  items: { name: string; value: string }[]
}

export interface StockAnalysis {
  type: string
  symbol: string
  name: string
  profileCode: string
  mode: string
  usedModels: string[]
  votes: { modelCode: string; direction: string; score: number; risk: string }[]
  final: { direction: string; score: number; risk: string; action: string; summary: string }
  cards: AnalysisCard[]
  summaries: { modelCode: string; summary: string }[]
  algorithmHits: { algorithmCode: string; pass: boolean; score: number; reason: string }[]
}

export interface HotNews {
  title: string
  summary: string
  source: string
  time: string
  tag: string
  url: string
  code: string
}

export interface HotTopic {
  topic: string
  heat: number
  newsCount: number
}

export interface HotBoard {
  code: string
  name: string
  changePercent: number
  change5: number
  leader: string
  leaderCode: string
  leaderChangePercent: number
}

export interface HotFeed {
  news: HotNews[]
  topics: HotTopic[]
  boards: HotBoard[]
}

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
