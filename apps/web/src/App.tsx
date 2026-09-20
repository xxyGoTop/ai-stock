import { Link, Navigate, Route, Routes } from 'react-router-dom'
import Home from './pages/Home'
import StockDetail from './pages/StockDetail'

export default function App() {
  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          AI Stock
        </Link>
        <span className="tag">Sprint 1 · 行情研究</span>
      </header>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/stock/:symbol" element={<StockDetail />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </div>
  )
}
