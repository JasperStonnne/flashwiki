import { Link, NavLink, Outlet } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'
import './admin-shell.css'

function navLinkClass({ isActive }: { isActive: boolean }): string {
  return isActive ? 'admin-nav-link admin-nav-link--active' : 'admin-nav-link'
}

export function AdminShell() {
  const { logout } = useAuth()

  return (
    <div className="admin-shell">
      <header className="topbar">
        <span className="topbar-brand">FPGWiki</span>
        <div className="topbar-actions">
          <button className="topbar-logout" type="button" onClick={logout}>
            退出登录
          </button>
        </div>
      </header>
      <div className="admin-body">
        <nav className="admin-nav">
          <h2 className="admin-nav-title">管理后台</h2>
          <NavLink className={navLinkClass} to="/admin/users">
            用户管理
          </NavLink>
          <NavLink className={navLinkClass} to="/admin/groups">
            权限组
          </NavLink>
          <Link className="admin-nav-back" to="/">
            ← 返回主页
          </Link>
        </nav>
        <div className="admin-content">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
