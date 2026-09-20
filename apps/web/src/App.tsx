import { Link, Navigate, NavLink, Route, Routes } from 'react-router-dom'
import Home from './pages/Home'
import Hot from './pages/Hot'
import Screening from './pages/Screening'
import Settings from './pages/Settings'
import StockDetail from './pages/StockDetail'

export default function App() {
  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          AI Stock
        </Link>
        <nav className="nav">
          <NavLink to="/" end>
            行情
          </NavLink>
          <NavLink to="/screening">选股</NavLink>
          <NavLink to="/hot">热点</NavLink>
          <NavLink to="/settings">设置</NavLink>
        </nav>
        <span className="tag">模拟盘 · 非实盘</span>
      </header>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/screening" element={<Screening />} />
        <Route path="/hot" element={<Hot />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/stock/:symbol" element={<StockDetail />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </div>
  )
}
