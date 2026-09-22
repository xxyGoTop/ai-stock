import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAnomalyRules, getMemory, listAgents, listAnalysisProfiles, listDailyPicks, listLlmModels, resetAnomalyRules, saveAnomalyRules, saveMemoryPreferences } from '@ai-stock/api-client'
import type { AgentPrompt, AnalysisProfile, AnomalyRules, DailyPickRecord, LlmModel, MemoryPreferences } from '@ai-stock/types'
import PaginatedPicks from '../components/PaginatedPicks'
import { PROFILE_KEY } from '../lib/cache'

const KIND_LABEL: Record<string, string> = {
  recommend: '今日推荐',
  screening: '今日选股',
  preopen: '盘前推荐',
  intraday: '盘中推荐',
  close_auction: '尾盘推荐',
  review: '收盘复盘',
}

export default function Settings() {
  const [profiles, setProfiles] = useState<AnalysisProfile[]>([])
  const [models, setModels] = useState<LlmModel[]>([])
  const [agents, setAgents] = useState<AgentPrompt[]>([])
  const [records, setRecords] = useState<DailyPickRecord[]>([])
  const [activeKey, setActiveKey] = useState('')
  const [current, setCurrent] = useState(localStorage.getItem(PROFILE_KEY) || 'stock_analysis_default')
  const [error, setError] = useState('')
  const [rules, setRules] = useState<AnomalyRules | null>(null)
  const [ruleMsg, setRuleMsg] = useState('')
  const [ruleBusy, setRuleBusy] = useState(false)
  const [prefs, setPrefs] = useState<MemoryPreferences>({ focusThemes: [], note: '' })
  const [themeInput, setThemeInput] = useState('')
  const [memMsg, setMemMsg] = useState('')
  const [memBusy, setMemBusy] = useState(false)

  useEffect(() => {
    Promise.all([listAnalysisProfiles(), listLlmModels(), listAgents(), listDailyPicks(), getAnomalyRules(), getMemory()])
      .then(([p, m, a, picks, cfg, mem]) => {
        setProfiles(p)
        setModels(m)
        setAgents(a)
        setRecords(picks)
        setRules(cfg.rules)
        setPrefs(mem.preferences || { focusThemes: [], note: '' })
        if (picks.length) {
          const prefer =
            picks.find((r) => r.kind === 'recommend') ||
            picks.find((r) => r.kind === 'screening') ||
            picks[0]
          setActiveKey(`${prefer.date}:${prefer.kind}`)
        }
      })
      .catch((err) => setError(err instanceof Error ? err.message : '加载配置失败'))
  }, [])

  function choose(code: string) {
    setCurrent(code)
    localStorage.setItem(PROFILE_KEY, code)
  }

  const active = useMemo(
    () => records.find((r) => `${r.date}:${r.kind}` === activeKey) || null,
    [records, activeKey],
  )

  return (
    <main>
      <section className="panel">
        <div className="header-line">
          <h2 className="section-title">设置</h2>
          <Link className="ghost-btn" to="/">
            返回对话
          </Link>
        </div>
      </section>

      <section className="panel" style={{ marginTop: 16 }}>
        <h2 className="section-title">每日推荐 / 选股记录</h2>
        <p className="muted">
          对话里生成「推荐」或「选股」时会按自然日落盘；同一天同类记录会覆盖。数据文件在服务端{' '}
          <code>services/api-go/data/daily_picks.json</code>。
        </p>
        {records.length === 0 ? (
          <p className="muted">还没有记录。回到对话点「推荐」或「选股」生成一次即可。</p>
        ) : (
          <>
            <div className="algo-pills">
              {records.map((r) => {
                const key = `${r.date}:${r.kind}`
                return (
                  <button
                    key={key}
                    type="button"
                    className={activeKey === key ? 'pill on' : 'pill'}
                    onClick={() => setActiveKey(key)}
                  >
                    {r.date} · {KIND_LABEL[r.kind] || r.title} · {r.count}只
                  </button>
                )
              })}
            </div>
            {active && (
              <div style={{ marginTop: 12 }}>
                <div className="header-line">
                  <div>
                    <strong>
                      {active.title} · {active.date}
                    </strong>
                    <div className="muted">
                      {active.asOf}
                      {active.summary ? ` · ${active.summary}` : ''}
                    </div>
                  </div>
                </div>
                <PaginatedPicks
                  items={active.picks}
                  pageSize={10}
                  onPick={() => undefined}
                  onAction={() => undefined}
                  showDetailLink
                />
              </div>
            )}
          </>
        )}
      </section>

      <section className="panel" style={{ marginTop: 16 }}>
        <h2 className="section-title">自选异动阈值</h2>
        <p className="muted">
          对话「异动」、后台 Notification Agent 和页内收件箱共用这套规则。交易日约每 2 分钟扫描一次，指纹去重后写入服务端收件箱。资金单位是亿元。改完立即生效，文件在{' '}
          <code>services/api-go/data/anomaly_rules.json</code>。
        </p>
        {rules ? (
          <>
            <div className="rule-grid">
              {(
                [
                  ['changeMild', '涨跌幅轻度 %', 0.1],
                  ['changeStrong', '涨跌幅强烈 %', 0.1],
                  ['volumeRatioMild', '量比轻度', 0.1],
                  ['volumeRatioStrong', '量比强烈', 0.1],
                  ['turnoverMild', '换手轻度 %', 0.1],
                  ['fundMildYi', '主力资金轻度 亿', 0.01],
                  ['fundStrongYi', '主力资金强烈 亿', 0.01],
                  ['fundPctMild', '资金占成交轻度 %', 0.1],
                  ['fundPctStrong', '资金占成交强烈 %', 0.1],
                  ['superMildYi', '超大单轻度 亿', 0.01],
                  ['liftMild', '拉升轻度 %', 0.1],
                  ['liftNotable', '拉升明显 %', 0.1],
                  ['liftStrong', '拉升强烈 %', 0.1],
                ] as [keyof AnomalyRules, string, number][]
              ).map(([key, label, step]) => (
                <label key={key} className="rule-field">
                  <span>{label}</span>
                  <input
                    type="number"
                    step={step}
                    value={rules[key]}
                    onChange={(e) => setRules({ ...rules, [key]: Number(e.target.value) })}
                  />
                </label>
              ))}
            </div>
            <div className="action-row" style={{ marginTop: 12 }}>
              <button
                type="button"
                className="ghost-btn"
                disabled={ruleBusy}
                onClick={() => {
                  setRuleBusy(true)
                  setRuleMsg('')
                  saveAnomalyRules(rules)
                    .then((cfg) => {
                      setRules(cfg.rules)
                      setRuleMsg('已保存，下次扫描异动按新阈值。')
                    })
                    .catch((err) => setRuleMsg(err instanceof Error ? err.message : '保存失败'))
                    .finally(() => setRuleBusy(false))
                }}
              >
                {ruleBusy ? '保存中…' : '保存阈值'}
              </button>
              <button
                type="button"
                className="ghost-btn"
                disabled={ruleBusy}
                onClick={() => {
                  setRuleBusy(true)
                  setRuleMsg('')
                  resetAnomalyRules()
                    .then((cfg) => {
                      setRules(cfg.rules)
                      setRuleMsg('已恢复默认阈值。')
                    })
                    .catch((err) => setRuleMsg(err instanceof Error ? err.message : '重置失败'))
                    .finally(() => setRuleBusy(false))
                }}
              >
                恢复默认
              </button>
            </div>
            {ruleMsg ? <p className="muted">{ruleMsg}</p> : null}
          </>
        ) : (
          <p className="muted">正在加载异动阈值…</p>
        )}
      </section>

      <section className="panel" style={{ marginTop: 16 }}>
        <h2 className="section-title">投研记忆</h2>
        <p className="muted">
          只保存你明确设置的关注方向；分析 / 板块选股会自动记入「近期研究」。自由对话时会把记忆注入模型上下文（不永久画像推断）。数据：
          <code>services/api-go/data/memory.json</code>。
        </p>
        <div className="algo-pills" style={{ marginTop: 8 }}>
          {(prefs.focusThemes || []).map((t) => (
            <button
              key={t}
              type="button"
              className="pill on"
              title="点击移除"
              onClick={() => setPrefs({ ...prefs, focusThemes: prefs.focusThemes.filter((x) => x !== t) })}
            >
              {t} ×
            </button>
          ))}
        </div>
        <div className="action-row" style={{ marginTop: 10, gap: 8, flexWrap: 'wrap' }}>
          <input
            value={themeInput}
            onChange={(e) => setThemeInput(e.target.value)}
            placeholder="添加题材，如 AI / 新能源"
            style={{ minWidth: 180, flex: 1 }}
            onKeyDown={(e) => {
              if (e.key !== 'Enter') return
              e.preventDefault()
              const t = themeInput.trim()
              if (!t || prefs.focusThemes.includes(t)) return
              setPrefs({ ...prefs, focusThemes: [...prefs.focusThemes, t].slice(0, 12) })
              setThemeInput('')
            }}
          />
          <button
            type="button"
            className="ghost-btn"
            onClick={() => {
              const t = themeInput.trim()
              if (!t || prefs.focusThemes.includes(t)) return
              setPrefs({ ...prefs, focusThemes: [...prefs.focusThemes, t].slice(0, 12) })
              setThemeInput('')
            }}
          >
            添加
          </button>
        </div>
        <label className="muted" style={{ display: 'block', marginTop: 12 }}>
          备注（可选）
          <textarea
            rows={2}
            value={prefs.note || ''}
            onChange={(e) => setPrefs({ ...prefs, note: e.target.value })}
            placeholder="例如：偏短线，少追高"
            style={{ display: 'block', width: '100%', marginTop: 6 }}
          />
        </label>
        <div className="action-row" style={{ marginTop: 12 }}>
          <button
            type="button"
            className="ghost-btn"
            disabled={memBusy}
            onClick={() => {
              setMemBusy(true)
              setMemMsg('')
              saveMemoryPreferences({ focusThemes: prefs.focusThemes, note: prefs.note || '' })
                .then((snap) => {
                  setPrefs(snap.preferences)
                  setMemMsg('已保存关注偏好。')
                })
                .catch((err) => setMemMsg(err instanceof Error ? err.message : '保存失败'))
                .finally(() => setMemBusy(false))
            }}
          >
            {memBusy ? '保存中…' : '保存偏好'}
          </button>
        </div>
        {memMsg ? <p className="muted">{memMsg}</p> : null}
      </section>

      <section className="panel" style={{ marginTop: 16 }}>
        <h2 className="section-title">分析设置</h2>
        <p className="muted">
          密钥只写项目根目录 <code>.env</code>，不入库。当前只用火山方舟：豆包 Seed 2.1 Pro → DeepSeek V4
          Flash → 量化规则。其他供应商先停用。
        </p>
        {error && <p className="warn">{error}</p>}
        <div className="algo-pills">
          {profiles.map((p) => (
            <button key={p.code} className={current === p.code ? 'pill on' : 'pill'} onClick={() => choose(p.code)}>
              {p.name} · {p.mode}
              {p.agentCode ? ` · ${p.agentCode}` : ''}
              {!p.ready ? '（未就绪）' : ''}
            </button>
          ))}
        </div>
      </section>
      <section className="panel" style={{ marginTop: 16 }}>
        <h3 style={{ marginTop: 0 }}>Agent Prompt</h3>
        <p className="muted">提示词在 <code>services/ai-python/prompts/</code>，一个目录一个 Agent，后续加分析员直接复制目录。</p>
        <div className="results">
          {agents.map((a) => (
            <div className="row" key={a.code}>
              <div>
                <strong>{a.name}</strong>
                <div className="muted">
                  {a.code} · {a.task} · {a.role} · v{a.version}
                </div>
                {a.description ? <div className="muted">{a.description}</div> : null}
              </div>
              <span className={a.enabled ? 'up' : 'muted'}>{a.enabled ? '已启用' : '停用'}</span>
            </div>
          ))}
        </div>
      </section>
      <section className="panel" style={{ marginTop: 16 }}>
        <h3 style={{ marginTop: 0 }}>模型目录</h3>
        <div className="results">
          {models.filter((m) => m.enabled).map((m) => (
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
