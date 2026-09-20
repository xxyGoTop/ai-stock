import { FormEvent, useEffect, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react'
import {
  companionChatStream,
  getCompanionBriefing,
  getIndicators,
  getKline,
  getStock,
  getWatchAnomalies,
} from '@ai-stock/api-client'
import type {
  AgentRunState,
  AgentStreamEvent,
  CompanionBlock,
  CompanionBriefing,
  CompanionChatResponse,
  CompanionWorkspace,
  IndicatorPoint,
  KlineBar,
  Quote,
} from '@ai-stock/types'
import ChatBlocks from '../components/ChatBlocks'
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
  listConversations,
  loadActiveConversation,
  saveActiveConversation,
  switchConversation,
  uid,
  type ChatMsg,
  type Conversation,
  type ResearchStep,
} from '../lib/chatSession'

const QUICK = [
  { label: '行情', message: '今天行情', action: 'market' },
  { label: '热点', message: '今日热点', action: 'hot' },
  { label: '选股', message: '帮我选股', action: 'screening' },
  { label: '推荐', message: '盘中推荐', action: 'intraday' },
  { label: '自选', message: '我的自选', action: 'watchlist' },
  { label: '异动', message: '看看自选异动', action: 'watch_anomaly' },
]

const ANOMALY_SEEN_KEY = 'ai-stock.anomaly.seen'
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
  const [bars, setBars] = useState<KlineBar[]>([])
  const [series, setSeries] = useState<IndicatorPoint[]>([])
  const listRef = useRef<HTMLDivElement>(null)
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
  const dragRef = useRef<{ startX: number; startW: number } | null>(null)

  function refreshList() {
    setConversations(listConversations())
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
      // 简报已展示的异动记为已读，避免随后主动推送重复
      if (data.anomalies?.length) {
        try {
          const raw = sessionStorage.getItem(ANOMALY_SEEN_KEY)
          const set = new Set(raw ? (JSON.parse(raw) as string[]) : [])
          data.anomalies.forEach((a) => a.fingerprint && set.add(a.fingerprint))
          sessionStorage.setItem(ANOMALY_SEEN_KEY, JSON.stringify([...set].slice(-80)))
        } catch {
          /* ignore */
        }
      }
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
    void bootstrap()
    return () => {
      abortRef.current?.abort()
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

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' })
  }, [messages, loading, liveProgress, liveText, liveBlocks, liveTools, agentState])

  // 自选异动：页内主动推送（去重指纹）
  useEffect(() => {
    let cancelled = false
    const seen = (): Set<string> => {
      try {
        const raw = sessionStorage.getItem(ANOMALY_SEEN_KEY)
        return new Set(raw ? (JSON.parse(raw) as string[]) : [])
      } catch {
        return new Set()
      }
    }
    const remember = (fps: string[]) => {
      const set = seen()
      fps.forEach((f) => set.add(f))
      const arr = [...set].slice(-80)
      try {
        sessionStorage.setItem(ANOMALY_SEEN_KEY, JSON.stringify(arr))
      } catch {
        /* ignore */
      }
    }
    const poll = async () => {
      if (cancelled || loading || document.hidden) return
      try {
        const scan = await getWatchAnomalies()
        if (cancelled || !scan.hasWatch || scan.count === 0) return
        const known = seen()
        const fresh = (scan.items || []).filter((it) => it.fingerprint && !known.has(it.fingerprint))
        if (!fresh.length) return
        remember(fresh.map((f) => f.fingerprint))
        const top = fresh[0]
        const msg: ChatMsg = {
          id: uid(),
          role: 'assistant',
          proactive: true,
          text: `【主动提醒】${scan.summary} 例如 ${top.name}：${(top.reasons || []).join('；')}。`,
          blocks: [
            {
              type: 'anomaly',
              title: '自选异动推送',
              text: scan.summary,
              items: fresh,
              meta: { asOf: scan.asOf, count: fresh.length },
            },
            {
              type: 'suggestions',
              title: '要不要继续',
              items: [`分析${top.name}`, '看看自选异动', '我的自选'],
            },
          ],
        }
        setMessages((prev) => [...prev, msg])
      } catch {
        /* ignore poll errors */
      }
    }
    const t0 = window.setTimeout(() => void poll(), 12_000)
    const timer = window.setInterval(() => void poll(), 180_000)
    return () => {
      cancelled = true
      window.clearTimeout(t0)
      window.clearInterval(timer)
    }
  }, [loading])

  useEffect(() => {
    if (workspace.type !== 'stock' || !workspace.symbol) {
      setQuote(null)
      setBars([])
      setSeries([])
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
    return () => {
      cancelled = true
    }
  }, [workspace.type, workspace.symbol])

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
    if (res.workspace) setWorkspace(res.workspace)
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    void send(input)
  }

  function onAction(action: string, symbol?: string, label?: string) {
    const map: Record<string, string> = {
      analyze: `分析 ${label || symbol || ''}`,
      watch: `把 ${label || symbol || ''} 加入自选`,
      paper: `模拟交易 ${label || symbol || ''}`,
      kline: `打开 ${label || symbol || ''} K线`,
      watchlist: '我的自选',
      watch_anomaly: '看看自选异动',
    }
    void send(map[action] || action, { symbol, action })
  }

  function onPickSymbol(symbol: string, name?: string) {
    void send(`分析 ${name || symbol}`, { symbol, action: 'analyze' })
  }

  function onCreateSession() {
    persistRef.current = false
    const conv = createConversation()
    hydrate(conv)
    refreshList()
    persistRef.current = true
    void bootstrap(true)
  }

  function onSelectSession(id: string) {
    if (id === activeId) return
    persistRef.current = false
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

        <div className="chat-stream" ref={listRef}>
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

        <form className="chat-composer" onSubmit={onSubmit}>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={
              loading
                ? '运行中可继续输入，发送将中断并开启新一轮…'
                : '问行情 / 热点 / 选股，或说「分析茅台」「加入自选」「打开K线」'
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
                    {(['overview', 'kline', 'analysis', 'paper'] as const).map((t) => (
                      <button
                        key={t}
                        type="button"
                        className={`pill ${tab === t ? 'on' : ''}`}
                        onClick={() => setWorkspace((w) => ({ ...w, tab: t }))}
                      >
                        {{ overview: '概览', kline: 'K线', analysis: '分析', paper: '模拟' }[t]}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </header>

            {workspace.type !== 'stock' && briefing && (
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

                {tab === 'analysis' && quote && (
                  <div className="workspace-block">
                    <p className="muted">完整分析结论在左侧对话中。可继续追问或切换到模拟交易。</p>
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

            {workspace.type === 'empty' && <p className="muted pad">从对话开始，研究目标会出现在这里。</p>}
          </>
        )}
      </aside>
    </main>
  )
}
