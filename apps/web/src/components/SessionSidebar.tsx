import type { Conversation } from '../lib/chatSession'
import { dayLabel } from '../lib/chatSession'

type Props = {
  conversations: Conversation[]
  activeId: string
  onSelect: (id: string) => void
  onCreate: () => void
  onDelete: (id: string) => void
}

export default function SessionSidebar({ conversations, activeId, onSelect, onCreate, onDelete }: Props) {
  const groups = groupByDay(conversations)

  return (
    <aside className="session-sidebar panel">
      <div className="session-side-head">
        <strong>会话</strong>
        <button type="button" className="ghost-btn" onClick={onCreate}>
          + 新建
        </button>
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
