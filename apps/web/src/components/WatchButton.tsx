import { MouseEvent, useEffect, useState } from 'react'
import { addWatchItem, getWatchlist, removeWatchItem } from '@ai-stock/api-client'
import { notifyWatchUpdated, WATCH_EVENT } from '../lib/watch'

export default function WatchButton({
  symbol,
  name,
  compact = false,
}: {
  symbol: string
  name?: string
  compact?: boolean
}) {
  const [on, setOn] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    const load = () => {
      getWatchlist()
        .then((data) => setOn(data.items.some((it) => it.symbol === symbol)))
        .catch(() => undefined)
    }
    load()
    window.addEventListener(WATCH_EVENT, load)
    return () => window.removeEventListener(WATCH_EVENT, load)
  }, [symbol])

  async function toggle(e: MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    setBusy(true)
    try {
      if (on) {
        await removeWatchItem(symbol)
        setOn(false)
      } else {
        await addWatchItem({ symbol, name })
        setOn(true)
      }
      notifyWatchUpdated()
    } catch (err) {
      window.alert(err instanceof Error ? err.message : '自选操作失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <button className={on ? 'watch-btn on' : 'watch-btn'} disabled={busy} onClick={toggle} type="button">
      {on ? (compact ? '已自选' : '取消自选') : compact ? '+ 自选' : '加入自选'}
    </button>
  )
}
