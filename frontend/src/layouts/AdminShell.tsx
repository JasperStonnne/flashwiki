import { Outlet } from 'react-router-dom'

export function AdminShell() {
  return (
    <div>
      <aside>
        <h1>Admin Shell Sidebar</h1>
      </aside>
      <Outlet />
    </div>
  )
}
