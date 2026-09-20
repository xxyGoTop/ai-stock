import type { ApiResponse, IndicatorResult, KlineBar, Quote, Stock } from '@ai-stock/types'

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
