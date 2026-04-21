import { Outlet } from 'react-router-dom'

export function MainShell() {
  return (
    <div>
      <header>
        <h1>Main Shell TopBar</h1>
      </header>
      <Outlet />
    </div>
  )
}
