import { Link, Outlet } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'
import { Sidebar } from '../components/Sidebar'
import './main-shell.css'

export function MainShell() {
  const { logout, userRole } = useAuth()

  return (
    <div className="main-shell">
      <header className="topbar">
        <span className="topbar-brand">FPGWiki</span>
        <div className="topbar-actions">
          <span className="topbar-avatar" aria-hidden="true" />
          <button className="topbar-logout" type="button" onClick={logout}>
            退出登录
          </button>
        </div>
      </header>
      <div className="main-body">
        <aside className="sidebar">
          <Sidebar />
          {userRole === 'manager' && (
            <Link className="sidebar-admin-link" to="/admin/users">
              <span className="sidebar-admin-icon" aria-hidden="true">
                ⚙
              </span>
              <span>管理后台</span>
            </Link>
          )}
        </aside>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
