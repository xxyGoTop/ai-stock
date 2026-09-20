import type {
  AlgorithmMeta,
  AnalysisProfile,
  ApiResponse,
  IndicatorResult,
  KlineBar,
  LlmModel,
  Quote,
  DailyNote,
  HotFeed,
  PaperAccount,
  ScreenResult,
  Stock,
  StockAnalysis,
} from '@ai-stock/types'

const API_BASE = (import.meta as { env?: { VITE_API_BASE?: string } }).env?.VITE_API_BASE || '/api/v1'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`)
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`)
  }
  const body = (await res.json()) as ApiResponse<T>
  if (body.code !== 0) {
    throw new Error(body.message || 'request failed')
  }
  return body.data
}

export function searchStocks(q: string) {
  return get<Stock[]>(`/stocks/search?q=${encodeURIComponent(q)}`)
}

export function getStock(symbol: string) {
  return get<Quote>(`/stocks/${encodeURIComponent(symbol)}`)
}

export function getQuote(symbol: string) {
  return get<Quote>(`/quotes?symbol=${encodeURIComponent(symbol)}`)
}

export function getKline(symbol: string, limit = 180) {
  return get<{ bars: KlineBar[]; freshness: IndicatorResult['freshness'] }>(
    `/kline?symbol=${encodeURIComponent(symbol)}&limit=${limit}`,
  )
}

export function getIndicators(symbol: string, limit = 180) {
  return get<IndicatorResult>(`/indicators?symbol=${encodeURIComponent(symbol)}&limit=${limit}`)
}

export function getIndexQuotes() {
  return get<Quote[]>('/quotes/indices')
}

export function listAlgorithms() {
  return get<{ items: AlgorithmMeta[] }>('/algorithms').then((d) => d.items)
}

export function runScreening(body: { detail?: number; limit?: number; algorithms?: string[] } = {}) {
  return post<ScreenResult>('/screening', body)
}

export function listLlmModels() {
  return get<{ items: LlmModel[] }>('/llm/models').then((d) => d.items)
}

export function listAnalysisProfiles() {
  return get<{ items: AnalysisProfile[] }>('/analysis-profiles').then((d) => d.items)
}

export function analyzeStock(symbol: string, profileCode?: string) {
  return post<StockAnalysis>('/ai/analyze', { symbol, profileCode })
}

export function getHotFeed(limit = 15) {
  return get<HotFeed>(`/hot?limit=${limit}`)
}

export function getDailyNote(symbol: string, force = false) {
  return get<DailyNote>(`/ai/daily-note?symbol=${encodeURIComponent(symbol)}${force ? '&force=1' : ''}`)
}

export function getPaperAccount() {
  return get<PaperAccount>('/paper/account')
}

export function placePaperOrder(body: { symbol: string; name?: string; side: 'buy' | 'sell'; price: number; qty: number }) {
  return post<PaperAccount>('/paper/orders', body)
}

export function resetPaperAccount() {
  return post<PaperAccount>('/paper/reset', {})
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`)
  }
  const payload = (await res.json()) as ApiResponse<T>
  if (payload.code !== 0) {
    throw new Error(payload.message || 'request failed')
  }
  return payload.data
}
