import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { RequireAuth } from './auth/RequireAuth'
import { RequireRole } from './auth/RequireRole'
import { AdminShell } from './layouts/AdminShell'
import { MainShell } from './layouts/MainShell'
import { AdminGroupsPage } from './pages/AdminGroupsPage'
import { AdminUsersPage } from './pages/AdminUsersPage'
import { DocPage } from './pages/DocPage'
import { Home } from './pages/Home'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />

        <Route
          element={
            <RequireAuth>
              <MainShell />
            </RequireAuth>
          }
        >
          <Route path="/" element={<Home />} />
          <Route path="/doc/:id" element={<DocPage />} />
        </Route>

        <Route
          element={
            <RequireRole role="manager">
              <AdminShell />
            </RequireRole>
          }
        >
          <Route path="/admin/users" element={<AdminUsersPage />} />
          <Route path="/admin/groups" element={<AdminGroupsPage />} />
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
