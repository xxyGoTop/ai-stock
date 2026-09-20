import type { CompanionBlock, CompanionBriefing, CompanionWorkspace } from '@ai-stock/types'

export type ChatMsg = {
  id: string
  role: 'assistant' | 'user'
  text: string
  blocks?: CompanionBlock[]
  progress?: ResearchStep[]
  tools?: { id: string; tool: string; title: string; status: 'running' | 'done' | 'error'; summary?: string }[]
  proactive?: boolean
}

export type ResearchStep = {
  id: string
  title: string
  status: 'pending' | 'running' | 'done' | 'error'
}

export type Conversation = {
  id: string
  title: string
  messages: ChatMsg[]
  workspace: CompanionWorkspace
  briefing: CompanionBriefing | null
  phaseLabel: string
  bootstrapped: boolean
  createdAt: number
  updatedAt: number
}

type Store = {
  activeId: string
  conversations: Conversation[]
}

const KEY = 'ai-stock.conversations.v2'
const LEGACY_KEY = 'ai-stock.chatSession'
const mem: { value: Store | null } = { value: null }

const defaultWorkspace: CompanionWorkspace = { type: 'market', tab: 'overview' }

export function uid() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

export function emptyConversation(partial?: Partial<Conversation>): Conversation {
  const now = Date.now()
  return {
    id: uid(),
    title: '新会话',
    messages: [],
    workspace: { ...defaultWorkspace },
    briefing: null,
    phaseLabel: '',
    bootstrapped: false,
    createdAt: now,
    updatedAt: now,
    ...partial,
  }
}

function readStore(): Store | null {
  try {
    const raw = sessionStorage.getItem(KEY)
    if (!raw) return null
    return JSON.parse(raw) as Store
  } catch {
    return null
  }
}

function writeStore(store: Store) {
  mem.value = store
  try {
    sessionStorage.setItem(KEY, JSON.stringify(store))
  } catch {
    /* ignore */
  }
}

function migrateLegacy(): Store | null {
  try {
    const raw = sessionStorage.getItem(LEGACY_KEY)
    if (!raw) return null
    const legacy = JSON.parse(raw) as {
      messages?: ChatMsg[]
      workspace?: CompanionWorkspace
      briefing?: CompanionBriefing | null
      phaseLabel?: string
      bootstrapped?: boolean
      updatedAt?: number
    }
    if (!legacy.messages?.length) return null
    const conv = emptyConversation({
      title: titleFromMessages(legacy.messages),
      messages: legacy.messages,
      workspace: legacy.workspace || { ...defaultWorkspace },
      briefing: legacy.briefing || null,
      phaseLabel: legacy.phaseLabel || '',
      bootstrapped: !!legacy.bootstrapped,
      updatedAt: legacy.updatedAt || Date.now(),
    })
    const store: Store = { activeId: conv.id, conversations: [conv] }
    writeStore(store)
    sessionStorage.removeItem(LEGACY_KEY)
    return store
  } catch {
    return null
  }
}

export function titleFromMessages(messages: ChatMsg[]): string {
  const user = messages.find((m) => m.role === 'user')
  if (user?.text) {
    const t = user.text.trim().replace(/\s+/g, ' ')
    return t.length > 18 ? `${t.slice(0, 18)}…` : t
  }
  const asst = messages.find((m) => m.role === 'assistant')
  if (asst?.text) {
    const t = asst.text.trim().replace(/\s+/g, ' ')
    return t.length > 18 ? `${t.slice(0, 18)}…` : t
  }
  return '新会话'
}

export function loadStore(): Store {
  if (mem.value?.conversations?.length) return mem.value
  const stored = readStore()
  if (stored?.conversations?.length) {
    mem.value = stored
    return stored
  }
  const migrated = migrateLegacy()
  if (migrated) return migrated
  const conv = emptyConversation()
  const store: Store = { activeId: conv.id, conversations: [conv] }
  writeStore(store)
  return store
}

export function loadActiveConversation(): Conversation {
  const store = loadStore()
  return store.conversations.find((c) => c.id === store.activeId) || store.conversations[0]
}

export function listConversations(): Conversation[] {
  return [...loadStore().conversations].sort((a, b) => b.updatedAt - a.updatedAt)
}

export function saveActiveConversation(patch: Partial<Conversation>) {
  const store = loadStore()
  const idx = store.conversations.findIndex((c) => c.id === store.activeId)
  if (idx < 0) return loadActiveConversation()
  const cur = store.conversations[idx]
  const next: Conversation = {
    ...cur,
    ...patch,
    updatedAt: Date.now(),
  }
  if (patch.messages) {
    next.title = titleFromMessages(patch.messages)
  }
  store.conversations[idx] = next
  writeStore({ ...store })
  return next
}

export function createConversation(): Conversation {
  const store = loadStore()
  // 若当前会话还是空的，直接复用，不堆空会话
  const active = store.conversations.find((c) => c.id === store.activeId)
  if (active && !active.bootstrapped && active.messages.length === 0) {
    return active
  }
  const conv = emptyConversation()
  store.conversations = [conv, ...store.conversations].slice(0, 30)
  store.activeId = conv.id
  writeStore(store)
  return conv
}

export function switchConversation(id: string): Conversation | null {
  const store = loadStore()
  const hit = store.conversations.find((c) => c.id === id)
  if (!hit) return null
  store.activeId = id
  writeStore(store)
  return hit
}

export function deleteConversation(id: string): Conversation {
  const store = loadStore()
  const nextList = store.conversations.filter((c) => c.id !== id)
  if (nextList.length === 0) {
    const conv = emptyConversation()
    writeStore({ activeId: conv.id, conversations: [conv] })
    return conv
  }
  const activeId = store.activeId === id ? nextList[0].id : store.activeId
  writeStore({ activeId, conversations: nextList })
  return nextList.find((c) => c.id === activeId) || nextList[0]
}

/** @deprecated use createConversation */
export function clearChatSession() {
  return createConversation()
}

/** @deprecated */
export function loadChatSession(): Conversation {
  return loadActiveConversation()
}

/** @deprecated */
export function saveChatSession(patch: Partial<Conversation>) {
  return saveActiveConversation(patch)
}

export function progressForAction(action?: string, message?: string): ResearchStep[] {
  const text = `${action || ''} ${message || ''}`
  if (/选股|screening/i.test(text)) {
    return [
      { id: 'plan', title: '制定选股方案', status: 'running' },
      { id: 'scan', title: '扫描候选池', status: 'pending' },
      { id: 'score', title: '五算法评分', status: 'pending' },
      { id: 'rank', title: '整理结果', status: 'pending' },
    ]
  }
  if (/热点|hot/i.test(text)) {
    return [
      { id: 'boards', title: '拉取热门板块', status: 'running' },
      { id: 'news', title: '整理快讯主题', status: 'pending' },
      { id: 'render', title: '生成热点摘要', status: 'pending' },
    ]
  }
  if (/推荐|盘前|盘中|尾盘|复盘|preopen|intraday|review/i.test(text)) {
    return [
      { id: 'boards', title: '扫描主流板块', status: 'running' },
      { id: 'leaders', title: '提取领涨股', status: 'pending' },
      { id: 'save', title: '写入今日记录', status: 'pending' },
    ]
  }
  if (/分析|analyze|研究|看看|茅台|股票|\d{6}/i.test(text)) {
    return [
      { id: 'quote', title: '获取实时行情', status: 'running' },
      { id: 'kline', title: '获取近期 K 线', status: 'pending' },
      { id: 'note', title: '生成每日笔记', status: 'pending' },
      { id: 'ai', title: 'AI 解读整理', status: 'pending' },
    ]
  }
  if (/行情|市场|briefing|market|北向/i.test(text)) {
    return [
      { id: 'index', title: '获取指数情况', status: 'running' },
      { id: 'nb', title: '北向资金', status: 'pending' },
      { id: 'boards', title: '主流板块', status: 'pending' },
      { id: 'summary', title: '生成今日摘要', status: 'pending' },
    ]
  }
  return [
    { id: 'think', title: '理解你的问题', status: 'running' },
    { id: 'tool', title: '调用研究工具', status: 'pending' },
    { id: 'render', title: '整理回复', status: 'pending' },
  ]
}

export function advanceProgress(steps: ResearchStep[]): ResearchStep[] {
  const next = steps.map((s) => ({ ...s }))
  const running = next.findIndex((s) => s.status === 'running')
  if (running >= 0) {
    next[running].status = 'done'
    if (running + 1 < next.length) next[running + 1].status = 'running'
  }
  return next
}

export function completeProgress(steps: ResearchStep[]): ResearchStep[] {
  return steps.map((s) => ({ ...s, status: 'done' as const }))
}

export function dayLabel(ts: number): string {
  const d = new Date(ts)
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const sameDay = d.toDateString() === now.toDateString()
  const yest = new Date(now)
  yest.setDate(now.getDate() - 1)
  if (sameDay) return '今天'
  if (d.toDateString() === yest.toDateString()) return '昨天'
  return `${d.getMonth() + 1}/${pad(d.getDate())}`
}
