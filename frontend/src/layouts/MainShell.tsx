import { useAuth } from '../auth/AuthContext'
import { Outlet } from 'react-router-dom'

export function MainShell() {
  const { logout } = useAuth()

  return (
    <div>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>Main Shell TopBar</h1>
        <button type="button" onClick={logout}>
          退出登录
        </button>
      </header>
      <Outlet />
    </div>
  )
}
