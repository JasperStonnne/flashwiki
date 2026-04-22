import { useAuth } from '../auth/AuthContext'
import { Outlet } from 'react-router-dom'

export function AdminShell() {
  const { logout } = useAuth()

  return (
    <div>
      <aside style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>Admin Shell Sidebar</h1>
        <button type="button" onClick={logout}>
          退出登录
        </button>
      </aside>
      <Outlet />
    </div>
  )
}
