import type { Conversation } from '../lib/chatSession'
import { dayLabel } from '../lib/chatSession'

type Props = {
  conversations: Conversation[]
  activeId: string
  collapsed?: boolean
  onToggleCollapse?: () => void
  onSelect: (id: string) => void
  onCreate: () => void
  onDelete: (id: string) => void
}

export default function SessionSidebar({
  conversations,
  activeId,
  collapsed,
  onToggleCollapse,
  onSelect,
  onCreate,
  onDelete,
}: Props) {
  const groups = groupByDay(conversations)

  if (collapsed) {
    return (
      <aside className="session-sidebar panel collapsed" title="会话栏已收起">
        <button type="button" className="side-rail-btn" onClick={onToggleCollapse} title="展开会话栏">
          ≫
        </button>
        <button type="button" className="side-rail-btn" onClick={onCreate} title="新建会话">
          +
        </button>
      </aside>
    )
  }

  return (
    <aside className="session-sidebar panel">
      <div className="session-side-head">
        <strong>会话</strong>
        <div className="session-side-actions">
          <button type="button" className="ghost-btn" onClick={onCreate}>
            + 新建
          </button>
          <button type="button" className="ghost-btn side-collapse-btn" onClick={onToggleCollapse} title="收起会话栏">
            ≪
          </button>
        </div>
      </div>
      <div className="session-list">
        {groups.map((g) => (
          <div key={g.label} className="session-group">
            <div className="session-day">{g.label}</div>
            {g.items.map((c) => (
              <div key={c.id} className={`session-item ${c.id === activeId ? 'active' : ''}`}>
                <button type="button" className="session-main" onClick={() => onSelect(c.id)} title={c.title}>
                  {c.title || '新会话'}
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
            ))}
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
