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
  agentCode?: string
  ready: boolean
  readyModels: string[]
}

export interface AgentPrompt {
  code: string
  name: string
  task: string
  role: string
  version: string
  enabled: boolean
  description: string
}

export interface LlmModel {
  code: string
  name: string
  providerCode: string
  roles: string[]
  costTier: string
  enabled: boolean
  ready: boolean
  exhausted?: boolean
  retryInSec?: number
  reason?: string
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
  profileName?: string
  agentCode?: string
  mode: string
  usedModels: string[]
  votes: { modelCode: string; direction: string; score: number; risk: string }[]
  final: { direction: string; score: number; risk: string; action: string; summary: string }
  cards: AnalysisCard[]
  summaries: { modelCode: string; summary: string }[]
  algorithmHits: { algorithmCode: string; short?: string; name?: string; pass: boolean; score: number; reason: string }[]
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

export interface DailyNote {
  type: string
  symbol: string
  name: string
  industry: string
  region?: string
  concepts?: string[]
  asOf: string
  price: number
  changePercent: number
  score: number
  primary: string
  primaryName: string
  strategies: { code: string; short: string; name: string; pass: boolean; score: number; reason: string }[]
  baseLabel: string
  baseLevel: string
  baseCount: number
  action: string
  actionLevel: 'buy' | 'wait' | 'watch' | string
  actionNote: string
  plan: {
    buyLow: number
    buyHigh: number
    stop: number
    target1: number
    target2: number
    stance: string
    entryType: string
    note: string
    direction?: 'buy' | 'watch' | string
    position?: number
    suggestedQty?: number
    riskLevel?: 'low' | 'medium' | 'high' | string
    invalidConditions?: string[]
  }
  reasons: string[]
  risks: string[]
  fund?: {
    status: string
    level: string
    text: string
    mainNetInflow: number | null
    mainNetInflowPct: number | null
  }
  capital?: {
    kind: 'inst' | 'hot' | 'mixed' | string
    kindLabel: string
    instText?: string
    text: string
    note?: string
  }
  chips?: {
    status: string
    text: string
    concentration90?: number | null
    profitRatio?: number | null
  }
  expect?: string
  verdict?: {
    macdTag?: string
    macdHot?: boolean
    items: { key: string; title: string; text: string; hint?: string; tone?: string }[]
    pros: string[]
    cons: string[]
  }
  cached?: boolean
}

export interface PaperPosition {
  symbol: string
  name: string
  qty: number
  available: number
  cost: number
  price: number
  marketValue: number
  pnl: number
}

export interface PaperOrder {
  id: string
  symbol: string
  name: string
  side: 'buy' | 'sell' | string
  price: number
  qty: number
  amount: number
  fee: number
  status: string
  createdAt: string
}

export interface PaperAccount {
  cash: number
  marketValue: number
  equity: number
  pnl: number
  pnlPct: number
  todayPnl?: number
  todayPnlPct?: number
  positions: PaperPosition[]
  orders: PaperOrder[]
}

export interface WatchItem {
  id: string
  symbol: string
  name: string
  market: Market | string
  sortOrder: number
  createdAt: string
  price?: number
  change?: number
  changePercent?: number
  industry?: string
}

export interface Watchlist {
  items: WatchItem[]
}

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
