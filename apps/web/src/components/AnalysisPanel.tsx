import { useEffect, useState } from 'react'
import { analyzeStock, listAnalysisProfiles } from '@ai-stock/api-client'
import type { AnalysisProfile, StockAnalysis } from '@ai-stock/types'
import { loadAnalysis, PROFILE_KEY, saveAnalysis } from '../lib/cache'

const DIR: Record<string, string> = { bullish: '偏多', bearish: '偏空', neutral: '中性' }

export default function AnalysisPanel({ symbol }: { symbol: string }) {
  const [profiles, setProfiles] = useState<AnalysisProfile[]>([])
  const [profile, setProfile] = useState(localStorage.getItem(PROFILE_KEY) || 'stock_analysis_default')
  const [data, setData] = useState<StockAnalysis | null>(() => loadAnalysis(symbol))
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setData(loadAnalysis(symbol))
    setError('')
  }, [symbol])

  useEffect(() => {
    listAnalysisProfiles()
      .then((items) => setProfiles(items))
      .catch(() => setProfiles([]))
  }, [])

  async function run() {
    setLoading(true)
    setError('')
    try {
      const next = await analyzeStock(symbol, profile)
      setData(next)
      saveAnalysis(symbol, next)
    } catch (err) {
      setError(err instanceof Error ? err.message : '分析失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="panel" style={{ marginTop: 16 }}>
      <div className="header-line wrap">
        <h2 className="section-title">AI 分析</h2>
        <div className="algo-pills tight">
          {profiles.map((p) => (
            <button
              key={p.code}
              className={profile === p.code ? 'pill on' : 'pill'}
              onClick={() => {
                setProfile(p.code)
                localStorage.setItem(PROFILE_KEY, p.code)
              }}
            >
              {p.name}
            </button>
          ))}
          <button className="ghost" disabled={loading} onClick={run}>
            {loading ? '分析中…' : data ? '重新分析' : '开始分析'}
          </button>
        </div>
      </div>
      {error && <p className="warn">{error}</p>}
      {data && (
        <div className="analysis">
          <div className={`verdict ${data.final.direction}`}>
            <strong>
              {DIR[data.final.direction] || data.final.direction} · {data.final.score} 分
            </strong>
            <span>{data.final.action}</span>
          </div>
          <p className="muted">{data.final.summary}</p>
          <div className="muted tiny">
            {data.profileCode} · {data.usedModels.join(' / ') || 'quant-rules'}
          </div>
          {data.algorithmHits?.length > 0 && (
            <div className="tag-row">
              {data.algorithmHits.map((h) => (
                <span key={h.algorithmCode} className={h.pass ? 'tag-chip accent' : 'tag-chip'}>
                  {h.algorithmCode}
                  {h.pass ? ' 命中' : ''}
                </span>
              ))}
            </div>
          )}
          {data.votes.length > 1 && (
            <div className="metrics">
              {data.votes.map((v) => (
                <div className="metric" key={v.modelCode}>
                  <span>{v.modelCode}</span>
                  {DIR[v.direction]} · {v.score}
                </div>
              ))}
            </div>
          )}
          <div className="cards">
            {data.cards.map((c) => (
              <article className="card" key={c.cardType + c.title}>
                <h3>
                  {c.title} <span className="muted">{c.score}</span>
                </h3>
                {c.items.map((it) => (
                  <div className="kv" key={it.name}>
                    <span className="muted">{it.name}</span>
                    <span>{it.value}</span>
                  </div>
                ))}
              </article>
            ))}
          </div>
        </div>
      )}
    </section>
  )
}
