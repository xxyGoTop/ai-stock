import { Link } from 'react-router-dom'
import { changeTone, formatChange } from '@ai-stock/business'
import type {
  CompanionBlock,
  HotBoard,
  HotNews,
  HotTopic,
  NorthboundFlow,
  Quote,
  RecommendPick,
  ScreenPick,
  SessionCard,
  WatchAnomaly,
  WatchItem,
} from '@ai-stock/types'
import PaginatedPicks from './PaginatedPicks'
import WatchPlanList from './WatchPlanList'

type Props = {
  blocks: CompanionBlock[]
  onAction: (action: string, symbol?: string, label?: string) => void
  onSuggest: (text: string) => void
  onPick: (symbol: string, name?: string) => void
}

export default function ChatBlocks({ blocks, onAction, onSuggest, onPick }: Props) {
  return (
    <div className="chat-blocks">
      {blocks.map((b, i) => (
        <BlockView key={`${b.type}-${i}`} block={b} onAction={onAction} onSuggest={onSuggest} onPick={onPick} />
      ))}
    </div>
  )
}

function BlockView({
  block,
  onAction,
  onSuggest,
  onPick,
}: {
  block: CompanionBlock
  onAction: Props['onAction']
  onSuggest: Props['onSuggest']
  onPick: Props['onPick']
}) {
  switch (block.type) {
    case 'text':
      return (
        <div className="cblock">
          {block.title && <h4>{block.title}</h4>}
          {block.text && <p>{block.text}</p>}
        </div>
      )
    case 'indices': {
      const items = (block.items as Quote[]) || []
      return (
        <div className="cblock">
          <h4>{block.title || '指数'}</h4>
          <div className="mini-grid">
            {items.map((q) => (
              <div key={q.symbol} className="mini-card">
                <span className="muted">{q.name}</span>
                <strong className={changeTone(q.changePercent)}>{formatChange(q.changePercent)}</strong>
              </div>
            ))}
          </div>
        </div>
      )
    }
    case 'northbound': {
      const nb = block.data as NorthboundFlow
      return (
        <div className="cblock">
          <h4>{block.title || '北向资金'}</h4>
          <p>{block.text || nb?.text}</p>
          {nb && (
            <p className="muted">
              净流入 {nb.netInflow.toFixed(1)} 亿 · 沪 {nb.shNetInflow.toFixed(1)} · 深 {nb.szNetInflow.toFixed(1)}
            </p>
          )}
        </div>
      )
    }
    case 'boards': {
      const items = (block.items as HotBoard[]) || []
      return (
        <div className="cblock">
          <h4>{block.title || '热门板块'}</h4>
          <div className="board-list compact">
            {items.slice(0, 8).map((b) => (
              <button
                type="button"
                className="board-row"
                key={b.code}
                onClick={() => b.leaderCode && onPick(b.leaderCode, b.leader)}
              >
                <span>{b.name}</span>
                <span className={changeTone(b.changePercent)}>{formatChange(b.changePercent)}</span>
                <span className="muted">{b.leader}</span>
              </button>
            ))}
          </div>
        </div>
      )
    }
    case 'topics': {
      const items = (block.items as HotTopic[]) || []
      return (
        <div className="cblock">
          <h4>{block.title || '热点'}</h4>
          <div className="tag-row">
            {items.slice(0, 8).map((t) => (
              <span className="chip" key={t.topic}>
                {t.topic}
              </span>
            ))}
          </div>
        </div>
      )
    }
    case 'news': {
      const items = (block.items as HotNews[]) || []
      return (
        <div className="cblock">
          <h4>{block.title || '快讯'}</h4>
          <ul className="news-mini">
            {items.slice(0, 5).map((n, idx) => (
              <li key={`${n.code}-${idx}`}>
                {n.url ? (
                  <a href={n.url} target="_blank" rel="noreferrer">
                    {n.title}
                  </a>
                ) : (
                  n.title
                )}
              </li>
            ))}
          </ul>
        </div>
      )
    }
    case 'session_cards': {
      const items = (block.items as SessionCard[]) || []
      return (
        <div className="cblock">
          <h4>{block.title || '陪伴节奏'}</h4>
          <div className="session-grid">
            {items.map((c) => (
              <button
                type="button"
                key={c.phase}
                className={`session-card ${c.active ? 'active' : ''}`}
                onClick={() => onSuggest(c.title)}
              >
                <strong>{c.title}</strong>
                <span className="muted">{c.summary}</span>
                {c.picks?.[0] && (
                  <span>
                    {c.picks[0].name} {formatChange(c.picks[0].changePercent)}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      )
    }
    case 'picks': {
      const items = (block.items as RecommendPick[]) || []
      const pageSize = Number(block.meta?.pageSize) || 10
      return (
        <div className="cblock">
          <h4>
            {block.title || '推荐'} <span className="muted">共 {items.length} 只</span>
          </h4>
          <PaginatedPicks items={items} pageSize={pageSize} onPick={onPick} onAction={onAction} />
        </div>
      )
    }
    case 'screen_picks': {
      const items = (block.items as ScreenPick[]) || []
      const data = block.data as { scanned?: number; qualified?: number } | undefined
      const pageSize = Number(block.meta?.pageSize) || 10
      return (
        <div className="cblock">
          <h4>
            {block.title || '选股结果'} <span className="muted">共 {items.length} 只</span>
          </h4>
          {data && (
            <p className="muted">
              扫描 {data.scanned ?? '-'} · 达标 {data.qualified ?? items.length}
            </p>
          )}
          <PaginatedPicks items={items} pageSize={pageSize} onPick={onPick} onAction={onAction} showDetailLink />
        </div>
      )
    }
    case 'stock_card': {
      const q = block.quote
      const symbol = block.symbol || q?.symbol
      const name = q?.name
      return (
        <div className="cblock stock-card-block">
          <h4>{block.title || '股票'}</h4>
          {block.text && <p>{block.text}</p>}
          {q && (
            <p>
              <strong>{q.name}</strong> <span className="muted">{q.symbol}</span>{' '}
              <span className={changeTone(q.changePercent)}>
                {q.price.toFixed(2)} {formatChange(q.changePercent)}
              </span>
            </p>
          )}
          {symbol && (
            <div className="action-row">
              {(block.actions || ['analyze', 'watch', 'paper', 'kline']).map((a) => (
                <button key={a} type="button" className="pill" onClick={() => onAction(a, symbol, name)}>
                  {{
                    analyze: '分析',
                    watch: '自选',
                    tomorrow_plan: '明日计划',
                    paper: '模拟',
                    kline: 'K线',
                  }[a] || a}
                </button>
              ))}
              <Link className="pill" to={`/stock/${symbol}`}>
                详情页
              </Link>
            </div>
          )}
        </div>
      )
    }
    case 'actions': {
      const symbol = block.symbol
      const labels = (block.items as string[]) || []
      const actions = block.actions || []
      return (
        <div className="cblock">
          <h4>{block.title || '操作'}</h4>
          <div className="action-row">
            {actions.map((a, i) => (
              <button key={a} type="button" className="pill" onClick={() => onAction(a, symbol)}>
                {labels[i] || a}
              </button>
            ))}
          </div>
        </div>
      )
    }
    case 'suggestions': {
      const items = (block.items as string[]) || []
      return (
        <div className="cblock">
          {block.title && <h4>{block.title}</h4>}
          <div className="tag-row">
            {items.map((s) => (
              <button type="button" className="chip clickable" key={s} onClick={() => onSuggest(s)}>
                {s}
              </button>
            ))}
          </div>
        </div>
      )
    }
    case 'watchlist': {
      const items = (block.items as WatchItem[]) || []
      const pageSize = Number(block.meta?.pageSize) || 8
      return (
        <div className="cblock">
          <h4>{block.title || '自选'}</h4>
          <WatchPlanList items={items} pageSize={pageSize} mode="watchlist" onPick={onPick} onAction={onAction} />
        </div>
      )
    }
    case 'tomorrow_plan': {
      const items = (block.items as WatchItem[]) || []
      const pageSize = Number(block.meta?.pageSize) || 6
      return (
        <div className="cblock">
          <h4>{block.title || '明日计划'}</h4>
          {block.text && <p className="muted">{block.text}</p>}
          <WatchPlanList
            items={items}
            pageSize={pageSize}
            mode="tomorrow_plan"
            onPick={onPick}
            onAction={onAction}
          />
        </div>
      )
    }
    case 'today_ops': {
      const items = (block.items as WatchItem[]) || []
      const pageSize = Number(block.meta?.pageSize) || 6
      return (
        <div className="cblock today-ops-block">
          <h4>{block.title || '今日操作'}</h4>
          {block.text && <p>{block.text}</p>}
          <WatchPlanList items={items} pageSize={pageSize} mode="today_ops" onPick={onPick} onAction={onAction} />
        </div>
      )
    }
    case 'anomaly': {
      const items = (block.items as WatchAnomaly[]) || []
      const levelLabel: Record<string, string> = {
        mild: '轻度',
        notable: '明显',
        strong: '强烈',
      }
      return (
        <div className="cblock anomaly-block">
          <h4>{block.title || '自选异动'}</h4>
          {block.text && <p>{block.text}</p>}
          <div className="anomaly-list">
            {items.map((it) => (
              <div className={`anomaly-row level-${it.level}`} key={it.fingerprint || it.symbol}>
                <div className="anomaly-head">
                  <button type="button" className="linkish" onClick={() => onPick(it.symbol, it.name)}>
                    <strong>
                      {it.name} <span className="muted">{it.symbol}</span>
                    </strong>
                  </button>
                  <span className={`anomaly-badge ${it.level}`}>{levelLabel[it.level] || it.level}</span>
                  <span className={changeTone(it.changePercent)}>{formatChange(it.changePercent)}</span>
                </div>
                <div className="anomaly-tags">
                  {it.fundText ? <span className="anomaly-tag fund">{it.fundText}</span> : null}
                  {it.liftText ? (
                    <span className={`anomaly-tag ${it.lift === 'lifting' ? 'lift' : 'flat'}`}>{it.liftText}</span>
                  ) : null}
                </div>
                <p className="muted anomaly-reasons">{(it.reasons || []).join(' · ')}</p>
                <div className="action-row tight">
                  <button type="button" className="pill" onClick={() => onAction('analyze', it.symbol, it.name)}>
                    继续研究
                  </button>
                  <button type="button" className="pill" onClick={() => onAction('kline', it.symbol, it.name)}>
                    K线
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )
    }
    case 'attribution': {
      const attr = (block.data || {}) as {
        primaryLabel?: string
        explanation?: string
        drivers?: { label?: string; detail?: string }[]
        stockChangePercent?: number
        industryChangePercent?: number
      }
      return (
        <div className="cblock attr-block">
          <h4>{block.title || '涨跌归因'}</h4>
          <p>
            <strong>{attr.primaryLabel || block.text || '综合判断'}</strong>
            {typeof attr.stockChangePercent === 'number' ? (
              <span className="muted"> · 个股 {attr.stockChangePercent >= 0 ? '+' : ''}{attr.stockChangePercent}%</span>
            ) : null}
            {typeof attr.industryChangePercent === 'number' ? (
              <span className="muted"> · 行业 {attr.industryChangePercent >= 0 ? '+' : ''}{attr.industryChangePercent}%</span>
            ) : null}
          </p>
          {(attr.explanation || block.text) && <p>{attr.explanation || block.text}</p>}
          {(attr.drivers || []).length > 0 && (
            <ul className="risk-points">
              {(attr.drivers || []).slice(0, 4).map((d, i) => (
                <li key={i}>
                  <strong>{d.label || '因素'}</strong>：{d.detail}
                </li>
              ))}
            </ul>
          )}
        </div>
      )
    }
    case 'risk': {
      const meta = (block.meta || {}) as { level?: string; action?: string; score?: number }
      const level = String(meta.level || 'medium')
      const points = Array.isArray(block.items)
        ? (block.items as unknown[]).map((x) => (typeof x === 'string' ? x : String((x as { text?: string })?.text || ''))).filter(Boolean)
        : []
      const levelLabel: Record<string, string> = { low: '偏低', medium: '中等', high: '偏高', notable: '需关注' }
      return (
        <div className={`cblock risk-block level-${level}`}>
          <div className="risk-head">
            <h4>{block.title || '风险提示'}</h4>
            <span className={`risk-badge ${level}`}>{levelLabel[level] || level}</span>
          </div>
          {block.text && <p>{block.text}</p>}
          {typeof meta.score === 'number' && meta.score > 0 ? (
            <p className="muted">综合分数 {meta.score}{meta.action ? ` · 建议 ${meta.action}` : ''}</p>
          ) : meta.action ? (
            <p className="muted">建议：{meta.action}</p>
          ) : null}
          {points.length > 0 && (
            <ul className="risk-points">
              {points.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          )}
          {block.symbol && (
            <div className="action-row tight">
              <button type="button" className="pill" onClick={() => onAction('analyze', block.symbol)}>
                再看分析
              </button>
              <button type="button" className="pill" onClick={() => onAction('kline', block.symbol)}>
                打开 K 线
              </button>
            </div>
          )}
        </div>
      )
    }
    case 'comparison': {
      const meta = (block.meta || {}) as {
        left?: { symbol?: string; name?: string; price?: number; changePercent?: number }
        right?: { symbol?: string; name?: string; price?: number; changePercent?: number }
      }
      const left = meta.left || {}
      const right = meta.right || {}
      const rows = Array.isArray(block.items)
        ? (block.items as { metric?: string; left?: string; right?: string; winner?: string }[])
        : []
      return (
        <div className="cblock comparison-block">
          <h4>{block.title || '股票对比'}</h4>
          {block.text && <p className="compare-summary">{block.text}</p>}
          <div className="compare-heads">
            <button type="button" className="compare-side" onClick={() => left.symbol && onPick(left.symbol, left.name)}>
              <strong>{left.name || 'A'}</strong>
              <span className="muted">{left.symbol}</span>
              {typeof left.price === 'number' && (
                <span className={changeTone(left.changePercent || 0)}>
                  {left.price.toFixed(2)} {formatChange(left.changePercent || 0)}
                </span>
              )}
            </button>
            <span className="compare-vs">VS</span>
            <button type="button" className="compare-side" onClick={() => right.symbol && onPick(right.symbol, right.name)}>
              <strong>{right.name || 'B'}</strong>
              <span className="muted">{right.symbol}</span>
              {typeof right.price === 'number' && (
                <span className={changeTone(right.changePercent || 0)}>
                  {right.price.toFixed(2)} {formatChange(right.changePercent || 0)}
                </span>
              )}
            </button>
          </div>
          <table className="compare-table">
            <thead>
              <tr>
                <th>维度</th>
                <th>{left.name || 'A'}</th>
                <th>{right.name || 'B'}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.metric}>
                  <td className="muted">{r.metric}</td>
                  <td className={r.winner === 'left' ? 'win' : undefined}>{r.left}</td>
                  <td className={r.winner === 'right' ? 'win' : undefined}>{r.right}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="action-row tight">
            {left.symbol && (
              <button type="button" className="pill" onClick={() => onAction('analyze', left.symbol, left.name)}>
                分析{left.name}
              </button>
            )}
            {right.symbol && (
              <button type="button" className="pill" onClick={() => onAction('analyze', right.symbol, right.name)}>
                分析{right.name}
              </button>
            )}
          </div>
        </div>
      )
    }
    case 'analysis':
    case 'daily_note':
      return (
        <div className="cblock">
          <h4>{block.title || (block.type === 'analysis' ? 'AI 解读' : '今日笔记')}</h4>
          <AnalysisDigest data={block.data} />
          {block.symbol && (
            <div className="action-row">
              <button type="button" className="pill" onClick={() => onAction('kline', block.symbol)}>
                打开 K 线
              </button>
              <button type="button" className="pill" onClick={() => onAction('watch', block.symbol)}>
                加入自选
              </button>
              <button type="button" className="pill" onClick={() => onAction('tomorrow_plan', block.symbol)}>
                加入明日计划
              </button>
              <button type="button" className="pill" onClick={() => onAction('paper', block.symbol)}>
                模拟交易
              </button>
              <Link className="pill" to={`/stock/${block.symbol}`}>
                个股详情
              </Link>
            </div>
          )}
        </div>
      )
    case 'kline':
      return (
        <div className="cblock">
          <h4>{block.title || 'K线'}</h4>
          <p>已在右侧研究工作台打开日 K。</p>
          {block.symbol && (
            <div className="action-row">
              <button type="button" className="pill" onClick={() => onAction('analyze', block.symbol)}>
                继续分析
              </button>
              <Link className="pill" to={`/stock/${block.symbol}`}>
                详情页
              </Link>
            </div>
          )}
        </div>
      )
    default:
      return block.text ? (
        <div className="cblock">
          <p>{block.text}</p>
        </div>
      ) : null
  }
}

function AnalysisDigest({ data }: { data: unknown }) {
  if (!data || typeof data !== 'object') return <p className="muted">暂无结构化结论</p>
  const d = data as Record<string, unknown>
  const final = d.final as { direction?: string; score?: number; risk?: string; action?: string; summary?: string } | undefined
  const attr =
    (d.attribution as { primaryLabel?: string; explanation?: string; drivers?: { label?: string; detail?: string }[] } | undefined) ||
    ((d.sector as { attribution?: { primaryLabel?: string; explanation?: string; drivers?: { label?: string; detail?: string }[] } } | undefined)
      ?.attribution)
  const sectorCards = ((d.cards as { cardType?: string; title?: string; items?: { name: string; value: string }[] }[]) || []).filter(
    (c) => c.cardType === 'sector' || c.cardType === 'attribution',
  )
  if (final?.summary) {
    return (
      <div>
        <p>
          <strong>
            {final.direction || ''} · 分数 {final.score ?? '-'} · 风险 {final.risk || '-'}
          </strong>
        </p>
        <p>{final.summary}</p>
        {final.action && <p className="muted">建议动作：{final.action}</p>}
        {attr?.primaryLabel || attr?.explanation ? (
          <div className="attr-box">
            <strong>涨跌归因 · {attr.primaryLabel || '综合'}</strong>
            {attr.explanation ? <p>{attr.explanation}</p> : null}
            {(attr.drivers || []).slice(0, 3).map((x, i) => (
              <p key={i} className="muted">
                [{x.label || '因素'}] {x.detail}
              </p>
            ))}
          </div>
        ) : null}
        {sectorCards.length > 0 ? (
          <div className="attr-cards">
            {sectorCards.map((c) => (
              <div key={(c.cardType || '') + (c.title || '')} className="attr-card">
                <strong>{c.title || c.cardType}</strong>
                {(c.items || []).slice(0, 4).map((it) => (
                  <div className="kv" key={it.name}>
                    <span className="muted">{it.name}</span>
                    <span>{it.value}</span>
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : null}
      </div>
    )
  }
  if (typeof d.actionNote === 'string') {
    return (
      <div>
        <p>
          <strong>{String(d.action || '')}</strong> {String(d.name || '')}
        </p>
        <p>{d.actionNote}</p>
      </div>
    )
  }
  return <p className="muted">已生成研究结论，可打开个股详情查看完整卡片。</p>
}
