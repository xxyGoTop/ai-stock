export function formatChange(percent: number) {
  const sign = percent > 0 ? '+' : ''
  return `${sign}${percent.toFixed(2)}%`
}

export function changeTone(percent: number): 'up' | 'down' | 'flat' {
  if (percent > 0) return 'up'
  if (percent < 0) return 'down'
  return 'flat'
}

export function padSymbol(symbol: string) {
  return String(symbol || '').replace(/\D/g, '').padStart(6, '0')
}
