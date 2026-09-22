import type { Conversation } from '../lib/chatSession'
import { dayLabel } from '../lib/chatSession'
import { IconPanelLeftCollapse, IconPanelLeftExpand, IconPlus } from './LayoutIcons'

type Props = {
  conversations: Conversation[]
  activeId: string
  collapsed?: boolean
  summaryTopics?: Record<string, string>
  onToggleCollapse?: () => void
  onSelect: (id: string) => void
  onCreate: () => void
  onDelete: (id: string) => void
}

export default function SessionSidebar({
  conversations,
  activeId,
  collapsed,
  summaryTopics,
  onToggleCollapse,
  onSelect,
  onCreate,
  onDelete,
}: Props) {
  const groups = groupByDay(conversations)

  if (collapsed) {
    return (
      <aside className="session-sidebar panel collapsed" aria-label="会话栏已收起">
        <button type="button" className="side-rail-btn" onClick={onToggleCollapse} title="展开会话栏">
          <IconPanelLeftExpand />
        </button>
        <button type="button" className="side-rail-btn" onClick={onCreate} title="新建会话">
          <IconPlus />
        </button>
      </aside>
    )
  }

  return (
    <aside className="session-sidebar panel">
      <div className="session-side-head">
        <strong>会话</strong>
        <div className="session-side-actions">
          <button type="button" className="icon-btn" onClick={onCreate} title="新建会话">
            <IconPlus />
          </button>
          <button type="button" className="icon-btn" onClick={onToggleCollapse} title="收起会话栏">
            <IconPanelLeftCollapse />
          </button>
        </div>
      </div>
      <div className="session-list">
        {groups.map((g) => (
          <div key={g.label} className="session-group">
            <div className="session-day">{g.label}</div>
            {g.items.map((c) => {
              const topic = summaryTopics?.[c.id]
              return (
                <div key={c.id} className={`session-item ${c.id === activeId ? 'active' : ''}`}>
                  <button type="button" className="session-main" onClick={() => onSelect(c.id)} title={topic || c.title}>
                    <span className="session-title">{c.title || '新会话'}</span>
                    {topic ? <span className="session-topic muted">{topic}</span> : null}
                  </button>
                  <button
                    type="button"
                    className="session-del"
                    title="删除"
                    onClick={(e) => {
                      e.stopPropagation()
                      if (window.confirm('删除这个会话？')) onDelete(c.id)
                    }}
                  >
                    ×
                  </button>
                </div>
              )
            })}
          </div>
        ))}
        {conversations.length === 0 && <p className="muted pad">暂无会话</p>}
      </div>
    </aside>
  )
}

function groupByDay(list: Conversation[]) {
  const map = new Map<string, Conversation[]>()
  for (const c of list) {
    const label = dayLabel(c.updatedAt || c.createdAt)
    if (!map.has(label)) map.set(label, [])
    map.get(label)!.push(c)
  }
  return [...map.entries()].map(([label, items]) => ({ label, items }))
}
