import { Link, Navigate, Route, Routes } from 'react-router-dom'
import TopTools from './components/TopTools'
import Chat from './pages/Chat'
import Paper from './pages/Paper'
import Settings from './pages/Settings'
import StockDetail from './pages/StockDetail'

export default function App() {
  return (
    <div className="app companion-app">
      <header className="topbar compact-topbar">
        <Link to="/" className="brand">
          AI 投研
        </Link>
        <span className="tag grow">对话优先 · 模拟盘</span>
        <TopTools />
      </header>
      <Routes>
        <Route path="/" element={<Chat />} />
        <Route path="/paper" element={<Paper />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/stock/:symbol" element={<StockDetail />} />
        <Route path="/market" element={<Navigate to="/" replace />} />
        <Route path="/screening" element={<Navigate to="/" replace />} />
        <Route path="/hot" element={<Navigate to="/" replace />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </div>
  )
}
