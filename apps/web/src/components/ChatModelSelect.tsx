import { useEffect, useMemo, useRef, useState } from 'react'
import type { LlmModel } from '@ai-stock/types'

const PROVIDER: Record<string, string> = {
  volcengine_ark: '火山方舟',
  'quant-local': '本地规则',
}

type Option = {
  code: string
  name: string
  hint: string
}

export default function ChatModelSelect({
  models,
  value,
  onChange,
}: {
  models: LlmModel[]
  value: string
  onChange: (code: string) => void
}) {
  const [open, setOpen] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)
  const ready = useMemo(
    () => models.filter((m) => m.enabled && m.ready && !m.exhausted),
    [models],
  )
  const options = useMemo<Option[]>(() => {
    const items: Option[] = [
      { code: 'auto', name: '自动回退', hint: '火山方舟优先' },
    ]
    for (const m of ready) {
      items.push({
        code: m.code,
        name: m.name || m.code,
        hint: PROVIDER[m.providerCode] || m.providerCode,
      })
    }
    return items
  }, [ready])
  const current = options.find((o) => o.code === value) || options[0]

  useEffect(() => {
    function onDoc(ev: MouseEvent) {
      if (!boxRef.current?.contains(ev.target as Node)) setOpen(false)
    }
    function onKey(ev: KeyboardEvent) {
      if (ev.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [])

  return (
    <div className={`chat-model-pick${open ? ' open' : ''}`} ref={boxRef}>
      <button
        type="button"
        className="chat-model-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        title="对话使用的模型"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="chat-model-kicker">{current.hint}</span>
        <span className="chat-model-name">{current.name}</span>
        <span className="chat-model-caret" aria-hidden>
          ▾
        </span>
      </button>
      {open ? (
        <ul className="chat-model-menu" role="listbox">
          {options.map((o) => (
            <li key={o.code}>
              <button
                type="button"
                role="option"
                aria-selected={o.code === current.code}
                className={o.code === current.code ? 'on' : ''}
                onClick={() => {
                  onChange(o.code)
                  setOpen(false)
                }}
              >
                <span>{o.name}</span>
                <em>{o.hint}</em>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}
