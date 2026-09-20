export const PAPER_EVENT = 'paper-updated'

export function notifyPaperUpdated() {
  window.dispatchEvent(new Event(PAPER_EVENT))
}
