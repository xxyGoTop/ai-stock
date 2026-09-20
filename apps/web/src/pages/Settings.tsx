import { useEffect, useState } from 'react'
import { listAnalysisProfiles, listLlmModels } from '@ai-stock/api-client'
import type { AnalysisProfile, LlmModel } from '@ai-stock/types'
import { PROFILE_KEY } from '../lib/cache'

export default function Settings() {
  const [profiles, setProfiles] = useState<AnalysisProfile[]>([])
  const [models, setModels] = useState<LlmModel[]>([])
  const [current, setCurrent] = useState(localStorage.getItem(PROFILE_KEY) || 'stock_analysis_default')
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([listAnalysisProfiles(), listLlmModels()])
      .then(([p, m]) => {
        setProfiles(p)
        setModels(m)
      })
      .catch((err) => setError(err instanceof Error ? err.message : '加载配置失败'))
  }, [])

  function choose(code: string) {
    setCurrent(code)
    localStorage.setItem(PROFILE_KEY, code)
  }

  return (
    <main>
      <section className="panel">
        <h2 className="section-title">分析设置</h2>
        <p className="muted">
          密钥只读环境变量，不入库。AIHubMix 用 <code>LLM_AIHUBMIX_KEY</code>（也认 <code>AIHUBMIX_API_KEY</code>），
          免费模型额度用完会自动切下一个：agents-a1-free → intern-s2-free → DeepSeek / Qwen → 量化规则。
        </p>
        <p className="muted tight">
          模型页：
          <a href="https://aihubmix.com/model/agents-a1-free" target="_blank" rel="noreferrer">
            Agents A1
          </a>
          {' · '}
          <a href="https://aihubmix.com/model/intern-s2-free" target="_blank" rel="noreferrer">
            Intern S2
          </a>
        </p>
        {error && <p className="warn">{error}</p>}
        <div className="algo-pills">
          {profiles.map((p) => (
            <button key={p.code} className={current === p.code ? 'pill on' : 'pill'} onClick={() => choose(p.code)}>
              {p.name} · {p.mode}
              {!p.ready ? '（未就绪）' : ''}
            </button>
          ))}
        </div>
      </section>
      <section className="panel" style={{ marginTop: 16 }}>
        <h3 style={{ marginTop: 0 }}>模型目录</h3>
        <div className="results">
          {models.map((m) => (
            <div className="row" key={m.code}>
              <div>
                <strong>{m.code}</strong>
                <div className="muted">
                  {m.providerCode} · {m.roles.join('/')} · {m.costTier}
                </div>
              </div>
              <span className={m.exhausted ? 'warn' : m.ready ? 'up' : 'muted'}>
                {m.exhausted ? `额度用尽，约 ${Math.ceil((m.retryInSec || 0) / 60)} 分钟后再试` : m.ready ? '可调用' : '缺密钥'}
              </span>
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}
