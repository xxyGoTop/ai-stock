import { useEffect, useState } from 'react'
import { changeTone, formatChange } from '@ai-stock/business'
import { getBoardDetail } from '@ai-stock/api-client'
import type { BoardDetail, BoardStock, CompanionWorkspace } from '@ai-stock/types'

type Props = {
  workspace: CompanionWorkspace
  onPick: (symbol: string, name?: string) => void
  onSuggest: (text: string) => void
}

export default function SectorWorkspace({ workspace, onPick, onSuggest }: Props) {
  const code = workspace.boardCode || workspace.symbol || ''
  const q = workspace.boardName || workspace.name || workspace.topic || ''
  const [detail, setDetail] = useState<BoardDetail | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    setDetail(null)
    getBoardDetail(code ? { code, limit: 30 } : { q, limit: 30 })
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [code, q])

  const board = detail?.board
  const stocks = detail?.stocks || []
  const title = board?.name || workspace.boardName || workspace.name || workspace.topic || '板块'
  const kind = workspace.type === 'topic' ? '题材' : '板块'

  return (
    <div className="workspace-sector">
      <div className="workspace-quote">
        <div>
          <h3>
            {title} <span className="muted">{kind}</span>
          </h3>
          {board ? (
            <p className={changeTone(board.changePercent)}>
              今日 {formatChange(board.changePercent)}
              <span className="muted"> · 近5日 {formatChange(board.change5)}</span>
            </p>
          ) : loading ? (
            <p className="muted">加载板块…</p>
          ) : (
            <p className="muted">{error || '暂无行情'}</p>
          )}
          {board?.leader ? (
            <p className="muted tiny">
              领涨{' '}
              <button
                type="button"
                className="linkish"
                onClick={() => board.leaderCode && onPick(board.leaderCode, board.leader)}
              >
                {board.leader}
              </button>{' '}
              {formatChange(board.leaderChangePercent)}
            </p>
          ) : null}
        </div>
        <div className="algo-pills tight">
          <button type="button" className="pill" onClick={() => onSuggest(`看看${title}板块`)}>
            刷新研究
          </button>
          <button type="button" className="pill on" onClick={() => onSuggest(`在${title}选股`)}>
            板块选股
          </button>
        </div>
      </div>

      <div className="workspace-block">
        <h3>
          成分观察 <span className="muted">{stocks.length} 只</span>
        </h3>
        {loading && !stocks.length ? <p className="muted">拉取成分股…</p> : null}
        {!loading && !stocks.length ? <p className="muted">暂无成分数据</p> : null}
        <div className="board-list">
          {stocks.map((st: BoardStock) => (
            <button type="button" className="board-row" key={st.symbol} onClick={() => onPick(st.symbol, st.name)}>
              <span>
                {st.name} <span className="muted">{st.symbol}</span>
              </span>
              <span className={changeTone(st.changePercent)}>{formatChange(st.changePercent)}</span>
              <span className="muted">{typeof st.price === 'number' ? st.price.toFixed(2) : st.price}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
