import type { AgentRunState } from '@ai-stock/types'
import type { ResearchStep } from '../lib/chatSession'

export type ToolLine = {
  id: string
  tool: string
  title: string
  status: 'running' | 'done' | 'error'
  summary?: string
}

const STATE_LABEL: Record<AgentRunState, string> = {
  idle: '待命',
  thinking: '理解问题',
  planning: '制定研究计划',
  tooling: '调用工具',
  streaming: '整理结论',
  done: '完成',
  error: '出错',
  cancelled: '已取消',
}

export function AgentStatusBar({
  state,
  intent,
}: {
  state: AgentRunState
  intent?: string
}) {
  if (state === 'idle' || state === 'done') return null
  return (
    <div className={`agent-status agent-status-${state}`}>
      <span className="agent-status-dot" />
      <span>{STATE_LABEL[state] || state}</span>
      {intent ? <span className="muted"> · {intent}</span> : null}
    </div>
  )
}

export default function ResearchProgress({
  steps,
  tools,
  compact,
}: {
  steps: ResearchStep[]
  tools?: ToolLine[]
  compact?: boolean
}) {
  if (!steps.length && !tools?.length) return null
  return (
    <div className={`research-progress ${compact ? 'compact' : ''}`}>
      {steps.length > 0 && (
        <>
          <div className="research-progress-title">研究计划</div>
          <ul>
            {steps.map((s) => (
              <li key={s.id} className={`rp-${s.status}`}>
                <span className="rp-mark">
                  {s.status === 'done' ? '✓' : s.status === 'running' ? '●' : s.status === 'error' ? '!' : '○'}
                </span>
                <span>{s.title}</span>
              </li>
            ))}
          </ul>
        </>
      )}
      {tools && tools.length > 0 && !compact && (
        <>
          <div className="research-progress-title" style={{ marginTop: 8 }}>
            Agent 工具
          </div>
          <ul>
            {tools.map((t) => (
              <li key={t.id} className={`rp-${t.status}`}>
                <span className="rp-mark">
                  {t.status === 'done' ? '✓' : t.status === 'running' ? '●' : '!'}
                </span>
                <span>
                  {t.title || t.tool}
                  {t.summary ? ` · ${t.summary}` : ''}
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}
