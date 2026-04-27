import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from 'react'

import { request } from '../api/client'
import { API_ENDPOINTS } from '../api/endpoints'
import type { ChangeRoleRequest, UserResponse, UserRole } from '../api/types'
import './admin-users.css'

type RowFeedback = {
  message: string
  type: 'success' | 'error'
}

type RowAction = 'role' | 'logout'

const avatarPalette = [
  'var(--primary)',
  'var(--green)',
  'var(--orange)',
  'var(--purple)',
  'var(--rose)',
  'var(--primary-dark)',
]

function formatDate(isoString: string): string {
  return isoString.slice(0, 10)
}

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }

  return '请求失败'
}

function avatarColor(userId: string): string {
  let hash = 0
  for (const char of userId) {
    hash += char.charCodeAt(0)
  }

  return avatarPalette[hash % avatarPalette.length]
}

function firstCharacter(value: string): string {
  const trimmed = value.trim()
  if (trimmed === '') {
    return '?'
  }

  return Array.from(trimmed)[0] ?? '?'
}

function roleTarget(role: UserRole): UserRole {
  return role === 'manager' ? 'member' : 'manager'
}

function roleActionLabel(role: UserRole): string {
  return role === 'manager' ? '撤销 Manager' : '册封 Manager'
}

function feedbackClassName(feedback: RowFeedback): string {
  return feedback.type === 'success'
    ? 'row-feedback row-feedback--success'
    : 'row-feedback row-feedback--error'
}

function clearBusyRow(setBusyRows: Dispatch<SetStateAction<Set<string>>>, userId: string) {
  setBusyRows((current) => {
    const next = new Set(current)
    next.delete(userId)
    return next
  })
}

function clearBusyAction(
  setBusyActions: Dispatch<SetStateAction<Record<string, RowAction>>>,
  userId: string,
) {
  setBusyActions((current) => {
    const next = { ...current }
    delete next[userId]
    return next
  })
}

function setBusyRow(setBusyRows: Dispatch<SetStateAction<Set<string>>>, userId: string) {
  setBusyRows((current) => {
    const next = new Set(current)
    next.add(userId)
    return next
  })
}

export function AdminUsersPage() {
  const feedbackTimersRef = useRef<Record<string, number>>({})
  const [busyActions, setBusyActions] = useState<Record<string, RowAction>>({})
  const [busyRows, setBusyRows] = useState<Set<string>>(new Set())
  const [currentUserId, setCurrentUserId] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [reloadKey, setReloadKey] = useState(0)
  const [rowFeedback, setRowFeedback] = useState<Record<string, RowFeedback>>({})
  const [users, setUsers] = useState<UserResponse[]>([])

  useEffect(() => {
    return () => {
      Object.values(feedbackTimersRef.current).forEach((timerId) => window.clearTimeout(timerId))
    }
  }, [])

  useEffect(() => {
    let mounted = true

    async function loadUsers() {
      setLoading(true)
      setLoadError(null)

      try {
        const [meData, usersData] = await Promise.all([
          request<UserResponse>('GET', API_ENDPOINTS.me),
          request<UserResponse[]>('GET', API_ENDPOINTS.adminUsers),
        ])
        if (!mounted) {
          return
        }

        setCurrentUserId(meData.id)
        setUsers(usersData)
      } catch (error) {
        if (mounted) {
          setLoadError(errorMessage(error))
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }

    loadUsers()

    return () => {
      mounted = false
    }
  }, [reloadKey])

  function clearFeedbackTimer(userId: string) {
    const timerId = feedbackTimersRef.current[userId]
    if (timerId === undefined) {
      return
    }

    window.clearTimeout(timerId)
    delete feedbackTimersRef.current[userId]
  }

  function clearRowFeedback(userId: string) {
    clearFeedbackTimer(userId)
    setRowFeedback((current) => {
      if (!(userId in current)) {
        return current
      }

      const next = { ...current }
      delete next[userId]
      return next
    })
  }

  function setRowError(userId: string, message: string) {
    clearFeedbackTimer(userId)
    setRowFeedback((current) => ({
      ...current,
      [userId]: { message, type: 'error' },
    }))
  }

  function setRowSuccess(userId: string, message: string) {
    clearFeedbackTimer(userId)
    setRowFeedback((current) => ({
      ...current,
      [userId]: { message, type: 'success' },
    }))
    feedbackTimersRef.current[userId] = window.setTimeout(() => {
      setRowFeedback((current) => {
        if (!(userId in current)) {
          return current
        }

        const next = { ...current }
        delete next[userId]
        return next
      })
      delete feedbackTimersRef.current[userId]
    }, 3000)
  }

  function beginRowAction(userId: string, action: RowAction) {
    clearRowFeedback(userId)
    setBusyRow(setBusyRows, userId)
    setBusyActions((current) => ({
      ...current,
      [userId]: action,
    }))
  }

  function finishRowAction(userId: string) {
    clearBusyRow(setBusyRows, userId)
    clearBusyAction(setBusyActions, userId)
  }

  async function handleChangeRole(userId: string, newRole: UserRole) {
    beginRowAction(userId, 'role')

    try {
      await request<null>(
        'PATCH',
        API_ENDPOINTS.adminUserRole(userId),
        { role: newRole } satisfies ChangeRoleRequest,
      )
      setUsers((current) =>
        current.map((user) => (user.id === userId ? { ...user, role: newRole } : user)),
      )
      setRowSuccess(userId, '角色已更新')
    } catch (error) {
      setRowError(userId, errorMessage(error))
    } finally {
      finishRowAction(userId)
    }
  }

  async function handleForceLogout(userId: string) {
    beginRowAction(userId, 'logout')

    try {
      await request<null>('POST', API_ENDPOINTS.adminUserForceLogout(userId))
      setRowSuccess(userId, '已强制下线')
    } catch (error) {
      setRowError(userId, errorMessage(error))
    } finally {
      finishRowAction(userId)
    }
  }

  function retry() {
    setReloadKey((current) => current + 1)
  }

  return (
    <div className="admin-users">
      <h1 className="admin-page-title">用户管理</h1>
      {loading && <div className="admin-empty">加载中...</div>}
      {loadError && !loading && (
        <div className="admin-error">
          <p>{loadError}</p>
          <button className="admin-btn-secondary" type="button" onClick={retry}>
            重试
          </button>
        </div>
      )}
      {!loading && !loadError && users.length === 0 && <div className="admin-empty">暂无用户</div>}
      {!loading && !loadError && users.length > 0 && (
        <UsersTable
          busyActions={busyActions}
          busyRows={busyRows}
          currentUserId={currentUserId}
          onChangeRole={handleChangeRole}
          onForceLogout={handleForceLogout}
          rowFeedback={rowFeedback}
          users={users}
        />
      )}
    </div>
  )
}

interface UsersTableProps {
  busyActions: Record<string, RowAction>
  busyRows: Set<string>
  currentUserId: string | null
  onChangeRole: (userId: string, newRole: UserRole) => void
  onForceLogout: (userId: string) => void
  rowFeedback: Record<string, RowFeedback>
  users: UserResponse[]
}

function UsersTable({
  busyActions,
  busyRows,
  currentUserId,
  onChangeRole,
  onForceLogout,
  rowFeedback,
  users,
}: UsersTableProps) {
  return (
    <div className="users-table-wrapper">
      <table className="users-table">
        <thead>
          <tr>
            <th aria-label="头像" />
            <th>邮箱</th>
            <th>显示名</th>
            <th>角色</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr className="users-row" key={user.id}>
              <td>
                <UserAvatar user={user} />
              </td>
              <td className="users-email">{user.email}</td>
              <td className="users-name">{user.display_name}</td>
              <td>
                <RoleBadge role={user.role} />
              </td>
              <td className="users-date">{formatDate(user.created_at)}</td>
              <td className="users-actions-cell">
                {user.id !== currentUserId && (
                  <div className="users-actions">
                    <button
                      className="admin-btn-secondary admin-btn-sm"
                      disabled={busyRows.has(user.id)}
                      onClick={() => onChangeRole(user.id, roleTarget(user.role))}
                      type="button"
                    >
                      {busyActions[user.id] === 'role' ? '处理中...' : roleActionLabel(user.role)}
                    </button>
                    <button
                      className="admin-btn-danger admin-btn-sm"
                      disabled={busyRows.has(user.id)}
                      onClick={() => onForceLogout(user.id)}
                      type="button"
                    >
                      {busyActions[user.id] === 'logout' ? '处理中...' : '强制下线'}
                    </button>
                  </div>
                )}
                {rowFeedback[user.id] && (
                  <span className={feedbackClassName(rowFeedback[user.id])}>
                    {rowFeedback[user.id].message}
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function UserAvatar({ user }: { user: UserResponse }) {
  return (
    <div className="user-avatar" style={{ background: avatarColor(user.id) }}>
      {firstCharacter(user.display_name)}
    </div>
  )
}

function RoleBadge({ role }: { role: UserRole }) {
  return (
    <span className={`role-badge role-badge--${role}`}>
      {role === 'manager' ? 'Manager' : 'Member'}
    </span>
  )
}
