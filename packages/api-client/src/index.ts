import type {
  AgentPrompt,
  AlgorithmMeta,
  AnalysisProfile,
  AnomalyScan,
  ApiResponse,
  CompanionBriefing,
  CompanionChatResponse,
  AgentStreamEvent,
  DailyPickRecord,
  IndicatorResult,
  KlineBar,
  LlmModel,
  Quote,
  DailyNote,
  HotFeed,
  NorthboundFlow,
  PaperAccount,
  TodayOpsScan,
  WatchItem,
  Watchlist,
  WatchTradePlan,
  ScreenResult,
  Stock,
  StockAnalysis,
  StockNewsFeed,
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

export function listAgents() {
  return get<{ items: AgentPrompt[] }>('/agents').then((d) => d.items)
}

export function analyzeStock(symbol: string, profileCode?: string, modelCode?: string) {
  return post<StockAnalysis>('/ai/analyze', { symbol, profileCode, modelCode })
}

export function getHotFeed(limit = 15) {
  return get<HotFeed>(`/hot?limit=${limit}`)
}

export function getStockNews(symbol: string, limit = 8) {
  return get<StockNewsFeed>(`/news?symbol=${encodeURIComponent(symbol)}&limit=${limit}`)
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

export function getWatchlist(category?: string) {
  const q = category ? `?category=${encodeURIComponent(category)}` : ''
  return get<Watchlist>(`/watchlist${q}`)
}

export function getWatchAnomalies() {
  return get<AnomalyScan>('/watchlist/anomalies')
}

export function getTodayOps() {
  return get<TodayOpsScan>('/watchlist/today-ops')
}

export function addWatchItem(body: {
  symbol: string
  name?: string
  market?: string
  category?: string
  planForDate?: string
  plan?: WatchTradePlan
}) {
  return post<WatchItem>('/watchlist/items', body)
}

export function updateWatchItem(
  symbol: string,
  body: { category?: string; planForDate?: string; plan?: WatchTradePlan; name?: string },
) {
  return patch<WatchItem>(`/watchlist/items/${encodeURIComponent(symbol)}`, body)
}

export function removeWatchItem(symbol: string) {
  return del<{ removed: string }>(`/watchlist/items/${encodeURIComponent(symbol)}`)
}

export function clearWatchlist() {
  return del<{ cleared: boolean }>('/watchlist')
}

export function getCompanionBriefing() {
  return get<CompanionBriefing>('/ai/companion/briefing')
}

export function companionChat(body: {
  message?: string
  symbol?: string
  action?: string
  profileCode?: string
  modelCode?: string
  messages?: { role: string; content: string }[]
}) {
  return post<CompanionChatResponse>('/ai/companion/chat', body)
}

/** Agent Run SSE：边跑边推 research.plan / tool.* / block / message.delta */
export async function companionChatStream(
  body: {
    message?: string
    symbol?: string
    action?: string
    profileCode?: string
    modelCode?: string
    messages?: { role: string; content: string }[]
  },
  handlers: {
    onEvent: (ev: AgentStreamEvent) => void
    signal?: AbortSignal
  },
): Promise<CompanionChatResponse> {
  const res = await fetch(`${API_BASE}/ai/companion/chat/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify(body),
    signal: handlers.signal,
  })
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`)
  }
  if (!res.body) {
    throw new Error('stream body empty')
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''
  let final: CompanionChatResponse = { reply: '', intent: '', blocks: [] }

  const dispatch = (event: string, raw: string) => {
    let data: unknown = {}
    try {
      data = raw ? JSON.parse(raw) : {}
    } catch {
      data = { message: raw }
    }
    const ev = { event, data } as AgentStreamEvent
    handlers.onEvent(ev)
    if (event === 'message.end') {
      const end = data as CompanionChatResponse & { ok?: boolean; reply?: string }
      final = {
        reply: end.reply || final.reply,
        intent: end.intent || final.intent,
        blocks: end.blocks || final.blocks,
        workspace: end.workspace || final.workspace,
      }
    }
    if (event === 'message.delta') {
      const d = data as { content?: string }
      if (d.content) final.reply = (final.reply || '') + d.content
    }
    if (event === 'block') {
      final.blocks = [...(final.blocks || []), data as CompanionChatResponse['blocks'][number]]
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    // SSE 事件以空行分隔
    let idx: number
    while ((idx = buffer.indexOf('\n\n')) >= 0) {
      const chunk = buffer.slice(0, idx)
      buffer = buffer.slice(idx + 2)
      const lines = chunk.split(/\r?\n/)
      let event = 'message'
      const dataLines: string[] = []
      for (const line of lines) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
      }
      dispatch(event, dataLines.join('\n'))
    }
  }

  return final
}

export function getNorthbound() {
  return get<NorthboundFlow>('/market/northbound')
}

export function listDailyPicks() {
  return get<{ items: DailyPickRecord[] }>('/daily-picks').then((d) => d.items || [])
}

export function getDailyPick(date?: string, kind = 'recommend') {
  const q = new URLSearchParams()
  if (date) q.set('date', date)
  if (kind) q.set('kind', kind)
  const qs = q.toString()
  return get<DailyPickRecord | null>(`/daily-picks${qs ? `?${qs}` : ''}`)
}

async function del<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { method: 'DELETE' })
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`)
  }
  const payload = (await res.json()) as ApiResponse<T>
  if (payload.code !== 0) {
    throw new Error(payload.message || 'request failed')
  }
  return payload.data
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

async function patch<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'PATCH',
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
