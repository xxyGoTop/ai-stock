import { useEffect, useState } from 'react'
import { analyzeStockStream, listAnalysisProfiles } from '@ai-stock/api-client'
import type { AnalysisProfile, StockAnalysis } from '@ai-stock/types'
import { loadAnalysis, PROFILE_KEY, saveAnalysis } from '../lib/cache'

const DIR: Record<string, string> = { bullish: '偏多', bearish: '偏空', neutral: '中性' }
const RISK: Record<string, string> = { low: '偏低', mid: '中性', high: '偏高' }
const ALGO: Record<string, string> = {
  year_high: '年新高',
  deep_rebound: '深调回升',
  forward_train: '火车轨',
  daily_observe: '每日观察',
  ma5_align: '五日线',
}
const MODEL: Record<string, string> = {
  'quant-rules': '量化规则',
  'deepseek-chat': 'DeepSeek',
  'qwen-plus': '通义千问',
  'doubao-seed-2-1-lite': '豆包 Seed 2.1 Lite',
  'doubao-seed-2-1-pro': '豆包 Seed 2.1 Pro',
  'deepseek-v4-flash': 'DeepSeek V4 Flash',
}
const PROFILE: Record<string, string> = {
  stock_analysis_fast: '快速（规则）',
  stock_analysis_default: '火山方舟（默认）',
  stock_analysis_ark: '火山方舟 Pro 优先',
  stock_analysis_ensemble: '方舟多模型综合',
}
const AGENT: Record<string, string> = {
  stock_analyst: '个股分析员',
  trading_planner: '交易计划员',
  screening_nl: '选股理解员',
  news_digest: '新闻摘要员',
  ensemble_judge: '综合裁判',
}

type ProgressLine = { step: string; title: string; status: string; summary?: string }

export default function AnalysisPanel({ symbol }: { symbol: string }) {
  const [profiles, setProfiles] = useState<AnalysisProfile[]>([])
  const [profile, setProfile] = useState(localStorage.getItem(PROFILE_KEY) || 'stock_analysis_default')
  const [data, setData] = useState<StockAnalysis | null>(() => loadAnalysis(symbol))
  const [loading, setLoading] = useState(false)
  const [progress, setProgress] = useState<ProgressLine[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    setData(loadAnalysis(symbol))
    setError('')
    setProgress([])
  }, [symbol])

  useEffect(() => {
    listAnalysisProfiles()
      .then((items) => setProfiles(items))
      .catch(() => setProfiles([]))
  }, [])

  async function run() {
    setLoading(true)
    setError('')
    setProgress([])
    try {
      const next = await analyzeStockStream(symbol, {
        profileCode: profile,
        onProgress: (p) => {
          setProgress((prev) => {
            const i = prev.findIndex((x) => x.step === p.step)
            if (i < 0) return [...prev, p]
            const copy = [...prev]
            copy[i] = p
            return copy
          })
        },
      })
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
      {loading && (
        <div className="analyze-progress">
          <p className="muted">正在分析，步骤会实时更新…</p>
          <ul>
            {progress.map((p) => (
              <li key={p.step} className={p.status === 'done' ? 'done' : 'running'}>
                <span className="ap-mark">{p.status === 'done' ? '✓' : '…'}</span>
                <span>
                  {p.title || p.step}
                  {p.summary ? <span className="muted"> · {p.summary}</span> : null}
                </span>
              </li>
            ))}
            {progress.length === 0 ? <li className="running">准备中…</li> : null}
          </ul>
        </div>
      )}
      {data && (
        <div className="analysis">
          <div className={`verdict ${data.final.direction}`}>
            <strong>
              {DIR[data.final.direction] || data.final.direction} · {data.final.score} 分
            </strong>
            <span>{data.final.action}</span>
          </div>
          <p className="muted">{data.final.summary}</p>
          {(data.attribution?.primaryLabel || data.attribution?.explanation) && (
            <div className="attr-box">
              <strong>涨跌归因 · {data.attribution.primaryLabel || '综合'}</strong>
              {data.attribution.explanation ? <p>{data.attribution.explanation}</p> : null}
              {(data.attribution.drivers || []).slice(0, 3).map((x, i) => (
                <p key={i} className="muted">
                  [{x.label || '因素'}] {x.detail}
                </p>
              ))}
            </div>
          )}
          <div className="muted tiny">
            {data.profileName || PROFILE[data.profileCode] || data.profileCode}
            {data.agentCode ? ` · ${AGENT[data.agentCode] || data.agentCode}` : ''}
            {' · '}
            {(data.usedModels.length ? data.usedModels : ['quant-rules']).map((m) => MODEL[m] || m).join(' / ')}
            {data.final.risk ? ` · 风险${RISK[data.final.risk] || data.final.risk}` : ''}
          </div>
          {data.algorithmHits?.length > 0 && (
            <div className="tag-row">
              {data.algorithmHits.map((h) => (
                <span key={h.algorithmCode} className={h.pass ? 'tag-chip accent' : 'tag-chip'}>
                  {h.short || ALGO[h.algorithmCode] || h.algorithmCode}
                  {h.pass ? ' 命中' : ' 未命中'}
                </span>
              ))}
            </div>
          )}
          {data.votes.length > 1 && (
            <div className="metrics">
              {data.votes.map((v) => (
                <div className="metric" key={v.modelCode}>
                  <span>{MODEL[v.modelCode] || v.modelCode}</span>
                  {DIR[v.direction]} · {v.score}
                </div>
              ))}
            </div>
          )}
          <div className="cards">
            {(data.cards || []).map((c) => (
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
