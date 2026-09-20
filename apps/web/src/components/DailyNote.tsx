import { useEffect, useState } from 'react'
import { getDailyNote } from '@ai-stock/api-client'
import type { DailyNote as Note } from '@ai-stock/types'
import TradingPlan from './TradingPlan'

const ACTION: Record<string, string> = {
  buy: '可买入',
  wait: '等待',
  watch: '观察',
}

export default function DailyNote({ symbol }: { symbol: string }) {
  const [note, setNote] = useState<Note | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    getDailyNote(symbol)
      .then((data) => {
        if (!cancelled) setNote(data)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '笔记加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [symbol])

  if (loading && !note) {
    return (
      <section className="panel note-panel">
        <h2 className="section-title">今日算法笔记</h2>
        <p className="muted">正在按五套算法生成当日分析…</p>
      </section>
    )
  }
  if (error && !note) {
    return (
      <section className="panel note-panel">
        <h2 className="section-title">今日算法笔记</h2>
        <p className="warn">{error}</p>
      </section>
    )
  }
  if (!note) return null
  const p = note.plan
  const v = note.verdict
  const industry = [note.industry || '未知行业', note.region].filter(Boolean).join(' · ')
  return (
    <section className="panel note-panel">
      <div className="header-line wrap">
        <div>
          <h2 className="section-title">今日算法笔记</h2>
          <p className="muted tight">
            {note.asOf} · {note.cached ? '当日已存档，不再重算' : '刚刚生成并保存'}
          </p>
        </div>
        <div className={`action-badge ${note.actionLevel}`}>{ACTION[note.actionLevel] || note.action}</div>
      </div>
      <div className="tag-row">
        <span className="tag-chip accent">主策略：{note.primaryName}</span>
        {note.strategies
          .filter((s) => s.pass)
          .map((s) => (
            <span className="tag-chip accent" key={s.code}>
              {s.short}
            </span>
          ))}
        {note.capital ? <span className={`tag-chip cap-${note.capital.kind}`}>{note.capital.kindLabel}</span> : null}
        <span className={`tag-chip base-${note.baseLevel}`}>{note.baseLabel}</span>
        <span className="tag-chip">分 {note.score}</span>
      </div>
      <div className="meta-board">
        {v?.macdTag ? <div className={`meta macd ${v.macdHot ? 'hot' : ''}`}>{v.macdTag}</div> : null}
        <div className="meta ind">
          <b>行业</b> {industry}
          {note.concepts?.length ? ` · ${note.concepts.slice(0, 4).join('、')}` : ''}
        </div>
        {note.fund ? (
          <div className={`meta fund ${note.fund.level}`}>
            <b>资金</b> {note.fund.text}
          </div>
        ) : null}
        {note.capital ? (
          <div className={`meta capital cap-${note.capital.kind}`}>
            <b>{note.capital.kindLabel}</b> {note.capital.instText || note.capital.text}
            {note.capital.note ? ` · ${note.capital.note}` : ''}
          </div>
        ) : null}
        {note.chips ? (
          <div className="meta chip">
            <b>筹码</b> {note.chips.text}
          </div>
        ) : null}
        {note.expect ? (
          <div className="meta expect">
            <b>预期</b> {note.expect}
          </div>
        ) : null}
      </div>
      <p className="muted">{note.actionNote}</p>
      <div className="plan-grid">
        <div className="plan buy">
          <span>买</span>
          <b>
            {p.buyLow.toFixed(2)} ~ {p.buyHigh.toFixed(2)}
          </b>
          <em>{p.entryType}</em>
        </div>
        <div className="plan sell">
          <span>止损</span>
          <b>{p.stop.toFixed(2)}</b>
          <em>破位离场</em>
        </div>
        <div className="plan take">
          <span>止盈</span>
          <b>{p.target1.toFixed(2)}</b>
          <em>先减半</em>
        </div>
        <div className="plan target">
          <span>目标</span>
          <b>{p.target2.toFixed(2)}</b>
          <em>余仓卖出</em>
        </div>
      </div>
      <p className="muted">{p.note}</p>
      <TradingPlan note={note} />
      {v ? (
        <div className="sv">
          <div className="sv-h">指标研判 · 综合强弱</div>
          {v.items.map((item) => (
            <div className={`sv-row ${item.tone || ''}`} key={item.key}>
              <b>{item.title}</b>
              {item.text}
              {item.hint ? <div className="sv-note">👉 {item.hint}</div> : null}
            </div>
          ))}
          <div className="sv-split">
            <div className="sv-box sv-pros">
              <h5>✅ 优点（强势点）</h5>
              <ul>
                {(v.pros.length ? v.pros : ['暂无足够强势点']).map((x) => (
                  <li key={x}>{x}</li>
                ))}
              </ul>
            </div>
            <div className="sv-box sv-cons">
              <h5>⚠️ 隐患（需要警惕）</h5>
              <ul>
                {(v.cons.length ? v.cons : ['暂无明显隐患']).map((x) => (
                  <li key={x}>{x}</li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      ) : null}
      <div className="note-cols">
        <div>
          <h3>买入理由</h3>
          {note.reasons.map((r) => (
            <p key={r}>{r}</p>
          ))}
        </div>
        <div>
          <h3>风险</h3>
          {(note.risks.length ? note.risks : ['暂无额外风险提示']).map((r) => (
            <p key={r}>{r}</p>
          ))}
        </div>
      </div>
    </section>
  )
}
