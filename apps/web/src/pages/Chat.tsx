import { FormEvent, useEffect, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react'
import {
  ackNotifications,
  companionChatStream,
  getCompanionBriefing,
  getIndicators,
  getKline,
  getMemory,
  getNotifications,
  getStock,
  getStockNews,
  listLlmModels,
} from '@ai-stock/api-client'
import type {
  AgentRunState,
  AgentStreamEvent,
  CompanionBlock,
  CompanionBriefing,
  CompanionChatResponse,
  CompanionNotice,
  CompanionWorkspace,
  IndicatorPoint,
  KlineBar,
  LlmModel,
  Quote,
  StockNewsFeed,
  WatchAnomaly,
  WatchItem,
} from '@ai-stock/types'
import ChatBlocks from '../components/ChatBlocks'
import ChatModelSelect from '../components/ChatModelSelect'
import KlineChart from '../components/KlineChart'
import {
  IconFullscreen,
  IconFullscreenExit,
  IconGrow,
  IconPanelLeftExpand,
  IconPanelRightCollapse,
  IconPanelRightExpand,
  IconShrink,
} from '../components/LayoutIcons'
import PaperTicket from '../components/PaperTicket'
import ResearchProgress, { AgentStatusBar, type ToolLine } from '../components/ResearchProgress'
import SessionSidebar from '../components/SessionSidebar'
import WatchButton from '../components/WatchButton'
import {
  createConversation,
  deleteConversation,
  flushConversationSync,
  hydrateFromServer,
  listConversations,
  loadActiveConversation,
  recordResearchRun,
  saveActiveConversation,
  switchConversation,
  uid,
  type ChatMsg,
  type Conversation,
  type ResearchStep,
} from '../lib/chatSession'
import { CHAT_MODEL_KEY } from '../lib/cache'

const QUICK = [
  { label: '行情', message: '今天行情', action: 'market' },
  { label: '热点', message: '今日热点', action: 'hot' },
  { label: '选股', message: '帮我选股', action: 'screening' },
  { label: '板块选股', message: '在半导体选股', action: 'screening' },
  { label: '推荐', message: '盘中推荐', action: 'intraday' },
  { label: '板块推荐', message: '在新能源推荐', action: 'recommend' },
  { label: '自选', message: '我的自选', action: 'watchlist' },
  { label: '明日计划', message: '明日计划', action: 'tomorrow_plan' },
  { label: '今日操作', message: '今日操作', action: 'today_ops' },
  { label: '对比', message: '茅台和比亚迪对比', action: 'compare' },
  { label: '异动', message: '看看自选异动', action: 'watch_anomaly' },
  { label: '关注', message: '我的关注', action: 'memory' },
]

const LAYOUT_KEY = 'ai-stock.layout.v1'
const WS_MIN = 280
const WS_MAX = 720
const WS_DEFAULT = 420

type LayoutPrefs = {
  sideCollapsed: boolean
  wsWidth: number
  wsCollapsed: boolean
  fullscreen: boolean
}

function loadLayout(): LayoutPrefs {
  try {
    const raw = localStorage.getItem(LAYOUT_KEY)
    if (!raw) return { sideCollapsed: false, wsWidth: WS_DEFAULT, wsCollapsed: false, fullscreen: false }
    const p = JSON.parse(raw) as Partial<LayoutPrefs>
    return {
      sideCollapsed: !!p.sideCollapsed,
      wsCollapsed: !!p.wsCollapsed,
      fullscreen: !!p.fullscreen,
      wsWidth: Math.min(WS_MAX, Math.max(WS_MIN, Number(p.wsWidth) || WS_DEFAULT)),
    }
  } catch {
    return { sideCollapsed: false, wsWidth: WS_DEFAULT, wsCollapsed: false, fullscreen: false }
  }
}

function saveLayout(p: LayoutPrefs) {
  try {
    localStorage.setItem(LAYOUT_KEY, JSON.stringify(p))
  } catch {
    /* ignore */
  }
}

function briefingFingerprints(data: CompanionBriefing): string[] {
  const out: string[] = []
  data.anomalies?.forEach((a) => {
    if (a.fingerprint) out.push(`a|${a.fingerprint}`)
  })
  data.todayOps?.forEach((it) => out.push(`t|${it.symbol}|${it.planForDate || ''}`))
  return out
}

function noticeFingerprints(n: CompanionNotice): string[] {
  const items = Array.isArray(n.items) ? n.items : []
  if (n.kind === 'today_ops') {
    return items.map((it) => {
      const row = it as WatchItem
      return `t|${row.symbol || ''}|${row.planForDate || ''}`
    })
  }
  return items
    .map((it) => {
      const row = it as WatchAnomaly
      return row.fingerprint ? `a|${row.fingerprint}` : ''
    })
    .filter(Boolean)
}

function noticeToMsg(n: CompanionNotice): ChatMsg {
  if (n.kind === 'today_ops') {
    const items = (Array.isArray(n.items) ? n.items : []) as WatchItem[]
    const topName = n.name || items[0]?.name || ''
    return {
      id: uid(),
      role: 'assistant',
      proactive: true,
      text: `【今日操作】${n.summary}`,
      blocks: [
        {
          type: 'today_ops',
          title: n.title || '今日操作推送',
          text: n.summary,
          items,
          meta: { ...(n.meta || {}), pageSize: 6, count: items.length },
        },
        {
          type: 'suggestions',
          title: '按计划执行',
          items: topName
            ? [`分析${topName}`, '今日操作', '明日计划', '我的自选']
            : ['今日操作', '明日计划', '我的自选'],
        },
      ],
    }
  }
  const items = (Array.isArray(n.items) ? n.items : []) as WatchAnomaly[]
  const top = items[0]
  return {
    id: uid(),
    role: 'assistant',
    proactive: true,
    text: `【主动提醒】${n.summary}`,
    blocks: [
      {
        type: 'anomaly',
        title: n.title || '自选异动推送',
        text: n.summary,
        items,
        meta: { ...(n.meta || {}), count: items.length },
      },
      {
        type: 'suggestions',
        title: '要不要继续',
        items: [`分析${top?.name || n.name || '这只股票'}`, '看看自选异动', '我的自选'],
      },
    ],
  }
}

function applyConversation(c: Conversation) {
  return {
    messages: c.messages,
    workspace: c.workspace,
    briefing: c.briefing,
    phaseLabel: c.phaseLabel,
  }
}

export default function Chat() {
  const first = loadActiveConversation()
  const [conversations, setConversations] = useState(() => listConversations())
  const [summaryTopics, setSummaryTopics] = useState<Record<string, string>>({})
  const [activeId, setActiveId] = useState(first.id)
  const [messages, setMessages] = useState<ChatMsg[]>(first.messages)
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [liveProgress, setLiveProgress] = useState<ResearchStep[]>([])
  const [liveTools, setLiveTools] = useState<ToolLine[]>([])
  const [liveText, setLiveText] = useState('')
  const [liveBlocks, setLiveBlocks] = useState<CompanionBlock[]>([])
  const [agentState, setAgentState] = useState<AgentRunState>('idle')
  const [agentIntent, setAgentIntent] = useState('')
  const [phaseLabel, setPhaseLabel] = useState(first.phaseLabel)
  const [workspace, setWorkspace] = useState<CompanionWorkspace>(first.workspace)
  const [briefing, setBriefing] = useState<CompanionBriefing | null>(first.briefing)
  const [quote, setQuote] = useState<Quote | null>(null)
  const [compareQuote, setCompareQuote] = useState<Quote | null>(null)
  const [bars, setBars] = useState<KlineBar[]>([])
  const [series, setSeries] = useState<IndicatorPoint[]>([])
  const [stockNews, setStockNews] = useState<StockNewsFeed | null>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const streamInnerRef = useRef<HTMLDivElement>(null)
  const pinBottomRef = useRef(true)
  const ignoreScrollRef = useRef(false)
  const persistRef = useRef(true)
  const abortRef = useRef<AbortController | null>(null)
  const runGenRef = useRef(0)
  const cancelIntentRef = useRef(false)
  const liveProgressRef = useRef<ResearchStep[]>([])
  const liveToolsRef = useRef<ToolLine[]>([])
  const layoutInit = loadLayout()
  const [sideCollapsed, setSideCollapsed] = useState(layoutInit.sideCollapsed)
  const [wsCollapsed, setWsCollapsed] = useState(layoutInit.wsCollapsed)
  const [wsWidth, setWsWidth] = useState(layoutInit.wsWidth)
  const [fullscreen, setFullscreen] = useState(layoutInit.fullscreen)
  const [showJump, setShowJump] = useState(false)
  const [chatModels, setChatModels] = useState<LlmModel[]>([])
  const [chatModel, setChatModel] = useState(() => localStorage.getItem(CHAT_MODEL_KEY) || 'auto')
  const dragRef = useRef<{ startX: number; startW: number } | null>(null)
  const briefingCoveredRef = useRef<Set<string>>(new Set())
  const deliveredNoticeRef = useRef<Set<string>>(new Set())

  useEffect(() => {
    listLlmModels()
      .then((items) => {
        const ready = items.filter((m) => m.enabled && m.ready && !m.exhausted)
        setChatModels(ready)
        setChatModel((cur) => (cur === 'auto' || ready.some((m) => m.code === cur) ? cur : 'auto'))
      })
      .catch(() => setChatModels([]))
  }, [])

  function chooseChatModel(code: string) {
    setChatModel(code)
    localStorage.setItem(CHAT_MODEL_KEY, code)
  }

  function refreshList() {
    setConversations(listConversations())
  }

  function refreshMemoryTopics() {
    void getMemory()
      .then((snap) => {
        const map: Record<string, string> = {}
        for (const s of snap.summaries || []) {
          if (s.conversationId && s.topic) map[s.conversationId] = s.topic
        }
        setSummaryTopics(map)
      })
      .catch(() => {
        /* ignore */
      })
  }

  function hydrate(c: Conversation) {
    const snap = applyConversation(c)
    setActiveId(c.id)
    setMessages(snap.messages)
    setWorkspace(snap.workspace)
    setBriefing(snap.briefing)
    setPhaseLabel(snap.phaseLabel)
  }

  async function bootstrap(force = false) {
    const cur = loadActiveConversation()
    if (!force && cur.bootstrapped && cur.messages.length > 0) {
      hydrate(cur)
      refreshList()
      return
    }
    setLoading(true)
    try {
      const data = await getCompanionBriefing()
      const msgs: ChatMsg[] = [
        { id: uid(), role: 'assistant', text: data.greeting, blocks: data.blocks },
      ]
      setBriefing(data)
      setPhaseLabel(data.phaseLabel)
      setMessages(msgs)
      setWorkspace({ type: 'market', tab: 'overview' })
      briefingCoveredRef.current = new Set(briefingFingerprints(data))
      saveActiveConversation({
        messages: msgs,
        briefing: data,
        phaseLabel: data.phaseLabel,
        workspace: { type: 'market', tab: 'overview' },
        bootstrapped: true,
      })
      refreshList()
    } catch (err) {
      const msgs: ChatMsg[] = [
        {
          id: uid(),
          role: 'assistant',
          text: `进房简报加载失败：${err instanceof Error ? err.message : '网络错误'}。你仍可以直接提问。`,
        },
      ]
      setMessages(msgs)
      saveActiveConversation({ messages: msgs, bootstrapped: true })
      refreshList()
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const store = await hydrateFromServer()
      if (cancelled) return
      const active =
        store.conversations.find((c) => c.id === store.activeId) || store.conversations[0]
      persistRef.current = false
      hydrate(active)
      refreshList()
      persistRef.current = true
      void bootstrap()
      refreshMemoryTopics()
    })()
    return () => {
      cancelled = true
      abortRef.current?.abort()
      // 跳转个股详情等会卸载 Chat；立刻落盘，避免 debounce 未完成时被旧快照盖掉
      void flushConversationSync()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!persistRef.current) return
    if (!messages.length && !briefing) return
    saveActiveConversation({
      messages,
      workspace,
      briefing,
      phaseLabel,
      bootstrapped: true,
    })
    refreshList()
  }, [messages, workspace, briefing, phaseLabel])

  function nearBottom(el: HTMLDivElement, slack = 120) {
    return el.scrollHeight - el.scrollTop - el.clientHeight < slack
  }

  function stickToBottom(force = false) {
    const el = listRef.current
    if (!el) return
    if (!force && !pinBottomRef.current) return
    ignoreScrollRef.current = true
    el.scrollTop = el.scrollHeight
    requestAnimationFrame(() => {
      el.scrollTop = el.scrollHeight
      ignoreScrollRef.current = false
    })
  }

  useEffect(() => {
    stickToBottom()
  }, [messages, loading, liveProgress, liveText, liveBlocks, liveTools, agentState])

  useEffect(() => {
    const inner = streamInnerRef.current
    if (!inner) return
    const ro = new ResizeObserver(() => stickToBottom())
    ro.observe(inner)
    return () => ro.disconnect()
  }, [])

  function onChatScroll() {
    if (ignoreScrollRef.current) return
    const el = listRef.current
    if (!el) return
    const pinned = nearBottom(el)
    pinBottomRef.current = pinned
    setShowJump(!pinned)
  }

  function jumpToBottom() {
    pinBottomRef.current = true
    setShowJump(false)
    stickToBottom(true)
  }

  // 收件箱：服务端 Notification Agent 扫描后写入，页内只拉未读
  useEffect(() => {
    let cancelled = false
    let busy = false
    const covered = (n: CompanionNotice) => {
      const fps = noticeFingerprints(n)
      if (!fps.length) return false
      const set = briefingCoveredRef.current
      return fps.every((fp) => set.has(fp))
    }
    const remember = (n: CompanionNotice) => {
      noticeFingerprints(n).forEach((fp) => briefingCoveredRef.current.add(fp))
    }
    const poll = async () => {
      if (cancelled || loading || document.hidden || busy) return
      busy = true
      try {
        const inbox = await getNotifications()
        const unread = inbox.filter((n) => n.id && !deliveredNoticeRef.current.has(n.id))
        if (cancelled || !unread.length) return
        const fresh = unread.filter((n) => !covered(n))
        unread.forEach((n) => deliveredNoticeRef.current.add(n.id))
        await ackNotifications(unread.map((n) => n.id))
        if (cancelled || !fresh.length) return
        fresh.forEach(remember)
        setMessages((prev) => [...prev, ...fresh.map(noticeToMsg)])
      } catch {
        /* ignore poll errors */
      } finally {
        busy = false
      }
    }
    const t0 = window.setTimeout(() => void poll(), 8_000)
    const timer = window.setInterval(() => void poll(), 45_000)
    const onVis = () => {
      if (!document.hidden) void poll()
    }
    document.addEventListener('visibilitychange', onVis)
    return () => {
      cancelled = true
      window.clearTimeout(t0)
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVis)
    }
  }, [loading])

  useEffect(() => {
    if (workspace.type === 'compare' && workspace.symbol && workspace.compareSymbol) {
      let cancelled = false
      setBars([])
      setSeries([])
      setStockNews(null)
      Promise.all([getStock(workspace.symbol), getStock(workspace.compareSymbol)])
        .then(([a, b]) => {
          if (cancelled) return
          setQuote(a)
          setCompareQuote(b)
        })
        .catch(() => undefined)
      return () => {
        cancelled = true
      }
    }
    setCompareQuote(null)
    if (workspace.type !== 'stock' || !workspace.symbol) {
      setQuote(null)
      setBars([])
      setSeries([])
      setStockNews(null)
      return
    }
    const symbol = workspace.symbol
    let cancelled = false
    getStock(symbol)
      .then((q) => {
        if (!cancelled) setQuote(q)
      })
      .catch(() => undefined)
    getKline(symbol, 180)
      .then((k) => {
        if (!cancelled) setBars(k.bars)
      })
      .catch(() => undefined)
    getIndicators(symbol, 180)
      .then((ind) => {
        if (!cancelled) setSeries(ind.series)
      })
      .catch(() => undefined)
    getStockNews(symbol, 8)
      .then((feed) => {
        if (!cancelled) setStockNews(feed)
      })
      .catch(() => {
        if (!cancelled) setStockNews({ symbol, news: [], notices: [] })
      })
    return () => {
      cancelled = true
    }
  }, [workspace.type, workspace.symbol, workspace.compareSymbol])

  function handleStreamEvent(ev: AgentStreamEvent) {
    switch (ev.event) {
      case 'message.start':
        setAgentState('thinking')
        setAgentIntent(ev.data.intent || '')
        break
      case 'research.plan': {
        const steps = (ev.data.steps || []).map((s) => ({
          id: s.id,
          title: s.title,
          status: (s.status as ResearchStep['status']) || 'pending',
        }))
        liveProgressRef.current = steps
        setLiveProgress(steps)
        const running = steps.some((s) => s.status === 'running')
        setAgentState(running ? 'planning' : 'tooling')
        break
      }
      case 'tool.start': {
        setAgentState('tooling')
        const id = `${ev.data.tool}-${Date.now()}`
        setLiveTools((prev) => {
          const next = [
            ...prev.map((t) => (t.status === 'running' ? { ...t, status: 'done' as const } : t)),
            { id, tool: ev.data.tool, title: ev.data.title || ev.data.tool, status: 'running' as const },
          ]
          liveToolsRef.current = next
          return next
        })
        break
      }
      case 'tool.result': {
        setLiveTools((prev) => {
          const next = [...prev]
          for (let i = next.length - 1; i >= 0; i--) {
            if (next[i].tool === ev.data.tool && next[i].status === 'running') {
              next[i] = {
                ...next[i],
                status: ev.data.ok === false ? 'error' : 'done',
                summary: ev.data.summary || ev.data.error,
              }
              break
            }
          }
          liveToolsRef.current = next
          return next
        })
        break
      }
      case 'block':
        setLiveBlocks((prev) => [...prev, ev.data as CompanionBlock])
        break
      case 'message.delta':
        setAgentState('streaming')
        setLiveText((prev) => prev + (ev.data.content || ''))
        break
      case 'error':
        setAgentState('error')
        setLiveText((prev) => prev || ev.data.message || '运行出错')
        break
      default:
        break
    }
  }

  async function send(message: string, extra?: { symbol?: string; action?: string }) {
    const text = message.trim()
    if (!text && !extra?.action) return
    // 新一轮推送：中断上一轮 Agent Run（不落「已取消」气泡）
    cancelIntentRef.current = false
    abortRef.current?.abort()
    const gen = ++runGenRef.current
    const ac = new AbortController()
    abortRef.current = ac
    pinBottomRef.current = true
    setShowJump(false)

    let nextMsgs = messages
    if (text) {
      nextMsgs = [...messages, { id: uid(), role: 'user', text }]
      setMessages(nextMsgs)
    }
    setInput('')
    setLoading(true)
    setAgentState('thinking')
    setAgentIntent(extra?.action || '')
    setLiveProgress([])
    setLiveTools([])
    setLiveText('')
    setLiveBlocks([])
    liveProgressRef.current = []
    liveToolsRef.current = []
    try {
      const res = await companionChatStream(
        {
          message: text || extra?.action || '',
          symbol: extra?.symbol,
          action: extra?.action,
          profileCode: chatModel === 'auto' ? 'stock_analysis_default' : undefined,
          modelCode: chatModel === 'auto' ? undefined : chatModel,
          messages: nextMsgs.slice(-10).map((m) => ({
            role: m.role === 'assistant' ? 'assistant' : 'user',
            content: m.text,
          })),
          conversationId: activeId,
        },
        {
          onEvent: (ev) => {
            if (gen !== runGenRef.current) return
            handleStreamEvent(ev)
          },
          signal: ac.signal,
        },
      )
      if (gen !== runGenRef.current) return
      applyResponse(res, nextMsgs, {
        progress: liveProgressRef.current,
        tools: liveToolsRef.current,
      })
      setAgentState('done')
      refreshMemoryTopics()
      void recordResearchRun({
        conversationId: activeId,
        intent: extra?.action || agentIntent || res.intent || '',
        progress: liveProgressRef.current,
        tools: liveToolsRef.current,
      })
    } catch (err) {
      if (gen !== runGenRef.current) return
      if ((err as Error)?.name === 'AbortError') {
        setAgentState('cancelled')
        if (cancelIntentRef.current) {
          setMessages((prev) => [
            ...prev,
            { id: uid(), role: 'assistant', text: '已取消本次 Agent Run。' },
          ])
        }
      } else {
        setAgentState('error')
        setMessages((prev) => [
          ...prev,
          {
            id: uid(),
            role: 'assistant',
            text: err instanceof Error ? err.message : '对话失败',
          },
        ])
      }
    } finally {
      if (gen === runGenRef.current) {
        setLiveProgress([])
        setLiveTools([])
        setLiveText('')
        setLiveBlocks([])
        setLoading(false)
        cancelIntentRef.current = false
        window.setTimeout(() => setAgentState('idle'), 400)
      }
    }
  }

  function cancelRun() {
    cancelIntentRef.current = true
    abortRef.current?.abort()
  }

  function applyResponse(
    res: CompanionChatResponse,
    base?: ChatMsg[],
    extras?: { progress?: ResearchStep[]; tools?: ToolLine[] },
  ) {
    const assistant: ChatMsg = {
      id: uid(),
      role: 'assistant',
      text: res.reply,
      blocks: res.blocks,
      progress: extras?.progress?.length
        ? extras.progress.map((s) => ({ ...s, status: s.status === 'error' ? 'error' : 'done' }))
        : undefined,
      tools: extras?.tools?.length
        ? extras.tools.map((t) => ({
            ...t,
            status: t.status === 'error' ? 'error' : 'done',
          }))
        : undefined,
    }
    setMessages(() => {
      const root = base || messages
      return [...root, assistant]
    })
    if (res.workspace) {
      const risk = res.blocks?.find((b) => b.type === 'risk')
      const meta = (risk?.meta || {}) as { level?: string; action?: string }
      const points = Array.isArray(risk?.items)
        ? (risk!.items as unknown[]).map((x) => (typeof x === 'string' ? x : '')).filter(Boolean)
        : []
      setWorkspace({
        ...res.workspace,
        riskLevel: meta.level || res.workspace.riskLevel,
        riskSummary: risk?.text || res.workspace.riskSummary,
        riskPoints: points.length ? points : res.workspace.riskPoints,
        riskAction: meta.action || res.workspace.riskAction,
      })
    }
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    void send(input)
  }

  function onAction(action: string, symbol?: string, label?: string) {
    const map: Record<string, string> = {
      analyze: `分析 ${label || symbol || ''}`,
      watch: `把 ${label || symbol || ''} 加入自选`,
      unwatch: `把 ${label || symbol || ''} 移出自选`,
      watch_clear: '清空自选',
      paper: `模拟交易 ${label || symbol || ''}`,
      kline: `打开 ${label || symbol || ''} K线`,
      watchlist: '我的自选',
      watch_anomaly: '看看自选异动',
      tomorrow_plan: `把 ${label || symbol || ''} 加入明日计划`,
      today_ops: '今日操作',
    }
    void send(map[action] || action, { symbol, action })
  }

  function onPickSymbol(symbol: string, name?: string) {
    void send(`分析 ${name || symbol}`, { symbol, action: 'analyze' })
  }

  function onCreateSession() {
    persistRef.current = false
    pinBottomRef.current = true
    setShowJump(false)
    const conv = createConversation()
    hydrate(conv)
    refreshList()
    persistRef.current = true
    void bootstrap(true)
  }

  function onSelectSession(id: string) {
    if (id === activeId) return
    persistRef.current = false
    pinBottomRef.current = true
    setShowJump(false)
    saveActiveConversation({ messages, workspace, briefing, phaseLabel, bootstrapped: true })
    const hit = switchConversation(id)
    if (hit) {
      hydrate(hit)
      if (!hit.bootstrapped || hit.messages.length === 0) {
        persistRef.current = true
        void bootstrap(true)
        return
      }
    }
    refreshList()
    persistRef.current = true
  }

  function onDeleteSession(id: string) {
    persistRef.current = false
    pinBottomRef.current = true
    setShowJump(false)
    const next = deleteConversation(id)
    hydrate(next)
    refreshList()
    persistRef.current = true
    if (!next.bootstrapped || next.messages.length === 0) {
      void bootstrap(true)
    }
  }

  const tab = workspace.tab || 'overview'

  useEffect(() => {
    saveLayout({ sideCollapsed, wsWidth, wsCollapsed, fullscreen })
  }, [sideCollapsed, wsWidth, wsCollapsed, fullscreen])

  useEffect(() => {
    document.body.classList.toggle('layout-fullscreen', fullscreen)
    return () => document.body.classList.remove('layout-fullscreen')
  }, [fullscreen])

  useEffect(() => {
    if (!fullscreen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setFullscreen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [fullscreen])

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      const d = dragRef.current
      if (!d) return
      // 拖拽条在工作台左侧：向左拖 → 工作台变宽
      const next = Math.min(WS_MAX, Math.max(WS_MIN, d.startW + (d.startX - e.clientX)))
      setWsWidth(next)
    }
    const onUp = () => {
      dragRef.current = null
      document.body.classList.remove('resizing-workspace')
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [])

  function startWsResize(e: ReactMouseEvent) {
    e.preventDefault()
    dragRef.current = { startX: e.clientX, startW: wsWidth }
    document.body.classList.add('resizing-workspace')
  }

  function shrinkWs() {
    if (wsCollapsed) return
    if (wsWidth <= WS_MIN + 20) {
      setWsCollapsed(true)
      return
    }
    setWsWidth((w) => Math.max(WS_MIN, w - 80))
  }

  function growWs() {
    if (wsCollapsed) {
      setWsCollapsed(false)
      return
    }
    setWsWidth((w) => Math.min(WS_MAX, w + 80))
  }

  const gridCols = [
    sideCollapsed ? '44px' : '200px',
    'minmax(0, 1fr)',
    wsCollapsed ? '44px' : `${wsWidth}px`,
  ].join(' ')

  return (
    <main
      className={`companion with-sessions${sideCollapsed ? ' side-collapsed' : ''}${wsCollapsed ? ' ws-collapsed' : ''}`}
      style={{ gridTemplateColumns: gridCols }}
    >
      <SessionSidebar
        conversations={conversations}
        activeId={activeId}
        collapsed={sideCollapsed}
        summaryTopics={summaryTopics}
        onToggleCollapse={() => setSideCollapsed((v) => !v)}
        onSelect={onSelectSession}
        onCreate={onCreateSession}
        onDelete={onDeleteSession}
      />

      <section className="companion-chat panel">
        <header className="companion-head">
          <div>
            <h1>投研伙伴</h1>
            <p className="muted">
              对话优先 · {phaseLabel || '陪伴中'}
              {briefing?.asOf ? ` · ${briefing.asOf}` : ''}
            </p>
          </div>
          <div className="head-actions">
            {sideCollapsed && (
              <button
                type="button"
                className="icon-btn"
                onClick={() => setSideCollapsed(false)}
                title="展开会话栏"
              >
                <IconPanelLeftExpand />
              </button>
            )}
            {wsCollapsed && (
              <button
                type="button"
                className="icon-btn"
                onClick={() => setWsCollapsed(false)}
                title="展开工作台"
              >
                <IconPanelRightExpand />
              </button>
            )}
            <button
              type="button"
              className="icon-btn"
              onClick={() => setFullscreen((v) => !v)}
              title={fullscreen ? '退出全屏 (Esc)' : '全屏'}
            >
              {fullscreen ? <IconFullscreenExit /> : <IconFullscreen />}
            </button>
            <button type="button" className="ghost-btn" disabled={loading} onClick={onCreateSession}>
              新会话
            </button>
            <button
              type="button"
              className="ghost-btn"
              disabled={loading}
              onClick={() => void send('今天行情', { action: 'briefing' })}
            >
              刷新简报
            </button>
          </div>
        </header>

        <div className="quick-bar">
          {QUICK.map((q) => (
            <button
              key={q.action}
              type="button"
              className="chip clickable"
              onClick={() => void send(q.message, { action: q.action })}
            >
              {q.label}
            </button>
          ))}
        </div>

        <div className="chat-stream-wrap">
        <div className="chat-stream" ref={listRef} onScroll={onChatScroll}>
          <div className="chat-stream-inner" ref={streamInnerRef}>
          {messages.map((m) => (
            <article key={m.id} className={`chat-bubble ${m.role}${m.proactive ? ' proactive' : ''}`}>
              <div className="chat-role">
                {m.role === 'assistant' ? (m.proactive ? '伙伴 · 主动提醒' : '伙伴') : '我'}
              </div>
              <p className="chat-text">{m.text}</p>
              {m.progress && m.progress.length > 0 && (
                <ResearchProgress steps={m.progress} tools={m.tools} compact />
              )}
              {m.blocks && m.blocks.length > 0 && (
                <ChatBlocks
                  blocks={m.blocks}
                  onAction={onAction}
                  onSuggest={(s) => void send(s)}
                  onPick={onPickSymbol}
                />
              )}
            </article>
          ))}
          {loading && (
            <article className="chat-bubble assistant agent-run">
              <div className="chat-role">伙伴 · Agent Run</div>
              <AgentStatusBar state={agentState} intent={agentIntent} />
              {liveText ? <p className="chat-text">{liveText}</p> : <p className="chat-text thinking">正在研究…</p>}
              <ResearchProgress steps={liveProgress} tools={liveTools} />
              {liveBlocks.length > 0 && (
                <ChatBlocks
                  blocks={liveBlocks}
                  onAction={onAction}
                  onSuggest={(s) => void send(s)}
                  onPick={onPickSymbol}
                />
              )}
              <div className="action-row" style={{ marginTop: 8 }}>
                <button type="button" className="ghost-btn" onClick={cancelRun}>
                  取消本次运行
                </button>
              </div>
            </article>
          )}
          </div>
        </div>
        {showJump && (
          <button type="button" className="chat-jump" onClick={jumpToBottom}>
            回到底部
          </button>
        )}
        </div>

        <form className="chat-composer" onSubmit={onSubmit}>
          <ChatModelSelect models={chatModels} value={chatModel} onChange={chooseChatModel} />
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={
              loading
                ? '运行中可继续输入，发送将中断并开启新一轮…'
                : '直接提问，或说「今天行情」「帮我选股」「分析茅台」'
            }
          />
          <button type="submit" disabled={!input.trim()}>
            {loading ? '打断并发送' : '发送'}
          </button>
        </form>
      </section>

      <aside className={`companion-workspace panel${wsCollapsed ? ' collapsed' : ''}`}>
        {wsCollapsed ? (
          <div className="ws-rail">
            <button type="button" className="side-rail-btn" onClick={() => setWsCollapsed(false)} title="展开工作台">
              <IconPanelRightExpand />
            </button>
            <button type="button" className="side-rail-btn" onClick={growWs} title="放大工作台">
              <IconGrow />
            </button>
            <button
              type="button"
              className="side-rail-btn"
              onClick={() => setFullscreen((v) => !v)}
              title={fullscreen ? '退出全屏' : '全屏'}
            >
              {fullscreen ? <IconFullscreenExit /> : <IconFullscreen />}
            </button>
          </div>
        ) : (
          <>
            <div
              className="workspace-resizer"
              role="separator"
              aria-orientation="vertical"
              aria-label="拖动调整工作台宽度"
              onMouseDown={startWsResize}
            />
            <header className="workspace-head">
              <div>
                <h2>研究工作台</h2>
                <p className="muted">
                  {workspace.type === 'stock'
                    ? `${workspace.name || quote?.name || ''} ${workspace.symbol || ''}`
                    : workspace.type === 'compare'
                      ? `${workspace.name || quote?.name || ''} VS ${workspace.compareName || compareQuote?.name || workspace.compareSymbol || ''}`
                      : '今日市场'}
                </p>
              </div>
              <div className="workspace-head-actions">
                <div className="ws-size-controls">
                  <button type="button" className="icon-btn" onClick={shrinkWs} title="缩小工作台">
                    <IconShrink />
                  </button>
                  <button type="button" className="icon-btn" onClick={growWs} title="放大工作台">
                    <IconGrow />
                  </button>
                  <button
                    type="button"
                    className="icon-btn"
                    onClick={() => setFullscreen((v) => !v)}
                    title={fullscreen ? '退出全屏 (Esc)' : '全屏'}
                  >
                    {fullscreen ? <IconFullscreenExit /> : <IconFullscreen />}
                  </button>
                  <button
                    type="button"
                    className="icon-btn"
                    onClick={() => setWsCollapsed(true)}
                    title="收起工作台"
                  >
                    <IconPanelRightCollapse />
                  </button>
                </div>
                {workspace.type === 'stock' && workspace.symbol && (
                  <div className="workspace-tabs">
                    {(['overview', 'kline', 'news', 'analysis', 'paper'] as const).map((t) => (
                      <button
                        key={t}
                        type="button"
                        className={`pill ${tab === t ? 'on' : ''}`}
                        onClick={() => setWorkspace((w) => ({ ...w, tab: t }))}
                      >
                        {{ overview: '概览', kline: 'K线', news: '新闻', analysis: '分析', paper: '模拟' }[t]}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </header>

            {workspace.type !== 'stock' && workspace.type !== 'compare' && briefing && (
              <div className="workspace-market">
                <div className="index-grid">
                  {briefing.indices.map((q) => (
                    <div className="index-card" key={q.symbol}>
                      <div className="muted">{q.name}</div>
                      <strong className={q.changePercent >= 0 ? 'up' : 'down'}>{q.price.toFixed(2)}</strong>
                      <span className={q.changePercent >= 0 ? 'up' : 'down'}>
                        {q.changePercent >= 0 ? '+' : ''}
                        {q.changePercent.toFixed(2)}%
                      </span>
                    </div>
                  ))}
                </div>
                {briefing.northbound && (
                  <div className="workspace-block">
                    <h3>北向资金</h3>
                    <p>{briefing.northbound.text}</p>
                    <p className="muted">
                      沪 {briefing.northbound.shNetInflow.toFixed(1)} 亿 · 深{' '}
                      {briefing.northbound.szNetInflow.toFixed(1)} 亿
                      {briefing.northbound.asOf ? ` · ${briefing.northbound.asOf}` : ''}
                    </p>
                  </div>
                )}
                <div className="workspace-block">
                  <h3>主流板块</h3>
                  <div className="board-list">
                    {briefing.boards.slice(0, 8).map((b) => (
                      <button
                        type="button"
                        className="board-row"
                        key={b.code}
                        onClick={() => b.leaderCode && onPickSymbol(b.leaderCode, b.leader)}
                      >
                        <span>{b.name}</span>
                        <span className={b.changePercent >= 0 ? 'up' : 'down'}>
                          {b.changePercent >= 0 ? '+' : ''}
                          {b.changePercent.toFixed(2)}%
                        </span>
                        <span className="muted">{b.leader}</span>
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {workspace.type === 'stock' && (
              <div className="workspace-stock">
                {quote && (
                  <div className="workspace-quote">
                    <div>
                      <h3>
                        {quote.name} <span className="muted">{quote.symbol}</span>
                      </h3>
                      <p className={quote.changePercent >= 0 ? 'up' : 'down'}>
                        {quote.price.toFixed(2)}{' '}
                        <span>
                          {quote.changePercent >= 0 ? '+' : ''}
                          {quote.changePercent.toFixed(2)}%
                        </span>
                      </p>
                    </div>
                    <WatchButton symbol={quote.symbol} name={quote.name} />
                  </div>
                )}

                {(tab === 'overview' || tab === 'kline') && bars.length > 0 && (
                  <div className="workspace-kline">
                    <KlineChart bars={bars} series={series} />
                  </div>
                )}

                {tab === 'news' && (
                  <div className="workspace-block">
                    <h3>相关新闻</h3>
                    {!stockNews ? (
                      <p className="muted">加载新闻…</p>
                    ) : stockNews.news.length === 0 ? (
                      <p className="muted">暂无相关新闻</p>
                    ) : (
                      <ul className="news-mini">
                        {stockNews.news.map((n, idx) => (
                          <li key={`${n.code || n.url}-${idx}`}>
                            {n.url ? (
                              <a href={n.url} target="_blank" rel="noreferrer">
                                {n.title}
                              </a>
                            ) : (
                              n.title
                            )}
                            <div className="muted tiny">
                              {[n.source, n.time].filter(Boolean).join(' · ')}
                            </div>
                          </li>
                        ))}
                      </ul>
                    )}
                    <h3 style={{ marginTop: 16 }}>近期公告</h3>
                    {!stockNews ? (
                      <p className="muted">加载公告…</p>
                    ) : !stockNews.notices.length ? (
                      <p className="muted">暂无公告</p>
                    ) : (
                      <ul className="news-mini">
                        {stockNews.notices.map((n, idx) => (
                          <li key={`${n.code || n.url}-n-${idx}`}>
                            {n.url ? (
                              <a href={n.url} target="_blank" rel="noreferrer">
                                {n.title}
                              </a>
                            ) : (
                              n.title
                            )}
                            <div className="muted tiny">
                              {[n.summary, n.time].filter(Boolean).join(' · ')}
                            </div>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )}

                {tab === 'analysis' && quote && (
                  <div className="workspace-block">
                    {workspace.riskSummary ? (
                      <div className={`ws-risk level-${workspace.riskLevel || 'medium'}`}>
                        <div className="risk-head">
                          <h3>风险提示</h3>
                          <span className={`risk-badge ${workspace.riskLevel || 'medium'}`}>
                            {{ low: '偏低', medium: '中等', high: '偏高', notable: '需关注' }[
                              workspace.riskLevel || 'medium'
                            ] || workspace.riskLevel}
                          </span>
                        </div>
                        <p>{workspace.riskSummary}</p>
                        {workspace.riskAction ? <p className="muted">建议：{workspace.riskAction}</p> : null}
                        {workspace.riskPoints && workspace.riskPoints.length > 0 ? (
                          <ul className="risk-points">
                            {workspace.riskPoints.map((p) => (
                              <li key={p}>{p}</li>
                            ))}
                          </ul>
                        ) : null}
                      </div>
                    ) : (
                      <p className="muted">完整分析结论在左侧对话中。可继续追问或切换到模拟交易。</p>
                    )}
                    <div className="action-row">
                      <button
                        type="button"
                        className="pill"
                        onClick={() => onAction('analyze', quote.symbol, quote.name)}
                      >
                        再分析一次
                      </button>
                      <button
                        type="button"
                        className="pill"
                        onClick={() => onAction('kline', quote.symbol, quote.name)}
                      >
                        看 K 线
                      </button>
                      <button
                        type="button"
                        className="pill"
                        onClick={() => onAction('paper', quote.symbol, quote.name)}
                      >
                        模拟交易
                      </button>
                    </div>
                  </div>
                )}

                {tab === 'paper' && quote && (
                  <PaperTicket symbol={quote.symbol} name={quote.name} price={quote.price} />
                )}
              </div>
            )}

            {workspace.type === 'compare' && (
              <div className="workspace-compare">
                <div className="compare-heads ws">
                  <button
                    type="button"
                    className="compare-side"
                    onClick={() => quote && onPickSymbol(quote.symbol, quote.name)}
                  >
                    <strong>{quote?.name || workspace.name || 'A'}</strong>
                    <span className="muted">{quote?.symbol || workspace.symbol}</span>
                    {quote && (
                      <span className={quote.changePercent >= 0 ? 'up' : 'down'}>
                        {quote.price.toFixed(2)} {quote.changePercent >= 0 ? '+' : ''}
                        {quote.changePercent.toFixed(2)}%
                      </span>
                    )}
                  </button>
                  <span className="compare-vs">VS</span>
                  <button
                    type="button"
                    className="compare-side"
                    onClick={() =>
                      (compareQuote || workspace.compareSymbol) &&
                      onPickSymbol(
                        compareQuote?.symbol || workspace.compareSymbol || '',
                        compareQuote?.name || workspace.compareName,
                      )
                    }
                  >
                    <strong>{compareQuote?.name || workspace.compareName || 'B'}</strong>
                    <span className="muted">{compareQuote?.symbol || workspace.compareSymbol}</span>
                    {compareQuote && (
                      <span className={compareQuote.changePercent >= 0 ? 'up' : 'down'}>
                        {compareQuote.price.toFixed(2)} {compareQuote.changePercent >= 0 ? '+' : ''}
                        {compareQuote.changePercent.toFixed(2)}%
                      </span>
                    )}
                  </button>
                </div>
                {quote && compareQuote && (
                  <table className="compare-table">
                    <tbody>
                      <tr>
                        <td className="muted">涨跌幅</td>
                        <td className={quote.changePercent >= compareQuote.changePercent ? 'win' : undefined}>
                          {quote.changePercent >= 0 ? '+' : ''}
                          {quote.changePercent.toFixed(2)}%
                        </td>
                        <td className={compareQuote.changePercent > quote.changePercent ? 'win' : undefined}>
                          {compareQuote.changePercent >= 0 ? '+' : ''}
                          {compareQuote.changePercent.toFixed(2)}%
                        </td>
                      </tr>
                      <tr>
                        <td className="muted">换手率</td>
                        <td>{quote.turnover?.toFixed(2) ?? '—'}%</td>
                        <td>{compareQuote.turnover?.toFixed(2) ?? '—'}%</td>
                      </tr>
                      <tr>
                        <td className="muted">量比</td>
                        <td>{quote.volumeRatio?.toFixed(2) ?? '—'}</td>
                        <td>{compareQuote.volumeRatio?.toFixed(2) ?? '—'}</td>
                      </tr>
                      <tr>
                        <td className="muted">行业</td>
                        <td>{quote.industry || '—'}</td>
                        <td>{compareQuote.industry || '—'}</td>
                      </tr>
                    </tbody>
                  </table>
                )}
                {workspace.riskSummary ? (
                  <div className={`ws-risk level-${workspace.riskLevel || 'notable'}`}>
                    <h3>风险提示</h3>
                    <p>{workspace.riskSummary}</p>
                  </div>
                ) : null}
                <div className="action-row">
                  {quote && (
                    <button type="button" className="pill" onClick={() => onAction('analyze', quote.symbol, quote.name)}>
                      分析{quote.name}
                    </button>
                  )}
                  {compareQuote && (
                    <button
                      type="button"
                      className="pill"
                      onClick={() => onAction('analyze', compareQuote.symbol, compareQuote.name)}
                    >
                      分析{compareQuote.name}
                    </button>
                  )}
                </div>
                <p className="muted">完整对比表与解读在左侧对话中。</p>
              </div>
            )}

            {workspace.type === 'empty' && <p className="muted pad">从对话开始，研究目标会出现在这里。</p>}
          </>
        )}
      </aside>
    </main>
  )
}
