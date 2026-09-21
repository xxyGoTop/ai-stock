import type { AlgorithmMeta, HotFeed, ScreenResult, Stock, StockAnalysis } from '@ai-stock/types'

export const PROFILE_KEY = 'ai-stock.analysisProfile'
export const CHAT_MODEL_KEY = 'ai-stock.chatModel'

export type ScreeningCache = {
  algos: AlgorithmMeta[]
  selected: string[]
  result: ScreenResult | null
  filter: string
  at: number
}

export type SearchCache = {
  q: string
  list: Stock[]
}

const screeningMem = { value: null as ScreeningCache | null }
const analysisMem = new Map<string, StockAnalysis>()

function readJSON<T>(key: string): T | null {
  try {
    const raw = sessionStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : null
  } catch {
    return null
  }
}

function writeJSON(key: string, value: unknown) {
  try {
    sessionStorage.setItem(key, JSON.stringify(value))
  } catch {
    /* ignore quota */
  }
}

export function loadScreening(): ScreeningCache | null {
  const stored = readJSON<ScreeningCache>('ai-stock.screening')
  if (!screeningMem.value) {
    screeningMem.value = stored
    return stored
  }
  if (stored && (stored.at || 0) > (screeningMem.value.at || 0)) {
    screeningMem.value = stored
  }
  return screeningMem.value
}

export function saveScreening(patch: Partial<ScreeningCache>) {
  const cur = loadScreening() || { algos: [], selected: [], result: null, filter: 'all', at: 0 }
  screeningMem.value = { ...cur, ...patch }
  writeJSON('ai-stock.screening', screeningMem.value)
}

export function loadAnalysis(symbol: string): StockAnalysis | null {
  const key = symbol.replace(/\D/g, '').padStart(6, '0')
  if (analysisMem.has(key)) return analysisMem.get(key) || null
  const stored = readJSON<StockAnalysis>(`ai-stock.analysis.${key}`)
  if (stored) analysisMem.set(key, stored)
  return stored
}

export function saveAnalysis(symbol: string, data: StockAnalysis) {
  const key = symbol.replace(/\D/g, '').padStart(6, '0')
  analysisMem.set(key, data)
  writeJSON(`ai-stock.analysis.${key}`, data)
}

export function loadSearch(): SearchCache {
  return readJSON<SearchCache>('ai-stock.search') || { q: '', list: [] }
}

export function saveSearch(q: string, list: Stock[]) {
  writeJSON('ai-stock.search', { q, list })
}

export function loadHot(): { feed: HotFeed; at: number } | null {
  return readJSON<{ feed: HotFeed; at: number }>('ai-stock.hot')
}

export function saveHot(feed: HotFeed) {
  writeJSON('ai-stock.hot', { feed, at: Date.now() })
}

export function formatClock(ts: number) {
  if (!ts) return ''
  const d = new Date(ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}
