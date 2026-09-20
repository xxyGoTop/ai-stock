export const WATCH_EVENT = 'watchlist-updated'

export function notifyWatchUpdated() {
  window.dispatchEvent(new Event(WATCH_EVENT))
}
