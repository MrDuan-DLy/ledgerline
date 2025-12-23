import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Transactions from './pages/Transactions';
import Import from './pages/Import';
import Receipts from './pages/Receipts';

function App() {
  return (
    <BrowserRouter>
      <div className="app-shell">
        <div className="backdrop" />
        <nav className="nav">
          <div className="nav-inner">
            <div className="brand">
              <span className="brand-mark" />
              <span className="brand-name">Ledgerline</span>
            </div>
            <div className="nav-links">
              <NavLink to="/" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
                Dashboard
              </NavLink>
              <NavLink to="/transactions" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
                Transactions
              </NavLink>
              <NavLink to="/import" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
                Import
              </NavLink>
              <NavLink to="/receipts" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
                Receipts
              </NavLink>
            </div>
          </div>
        </nav>

        <main className="container">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/transactions" element={<Transactions />} />
            <Route path="/import" element={<Import />} />
            <Route path="/receipts" element={<Receipts />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
