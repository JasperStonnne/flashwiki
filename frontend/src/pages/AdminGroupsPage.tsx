import { useEffect, useState, type MouseEvent as ReactMouseEvent } from 'react'
import { createPortal } from 'react-dom'

import { request } from '../api/client'
import { API_ENDPOINTS } from '../api/endpoints'
import type { AddMemberRequest, CreateGroupRequest, GroupListItem, GroupMemberResponse, GroupResponse, UpdateGroupRequest, UserResponse } from '../api/types'
import './admin-groups.css'

const AVATAR_PALETTE = ['var(--primary)', 'var(--green)', 'var(--orange)', 'var(--purple)', 'var(--rose)', 'var(--primary-dark)']

type CardMode = 'view' | 'renaming' | 'changing-leader' | 'confirming-delete'

function avatarColor(id: string): string {
  let hash = 0
  for (const char of id) {
    hash += char.charCodeAt(0)
  }
  return AVATAR_PALETTE[hash % AVATAR_PALETTE.length]
}

function firstChar(value: string): string {
  const trimmed = value.trim()
  if (trimmed === '') {
    return '?'
  }
  return Array.from(trimmed)[0] ?? '?'
}

function formatDate(iso: string): string { return iso.slice(0, 10) }

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }
  return '请求失败'
}

function findUser(users: UserResponse[], userId: string): UserResponse | undefined { return users.find((user) => user.id === userId) }

function hasMember(members: GroupMemberResponse[] | null, userId: string): boolean { return members?.some((member) => member.user_id === userId) ?? false }

function toLocalMember(groupId: string, userId: string): GroupMemberResponse { return { group_id: groupId, user_id: userId, joined_at: new Date().toISOString() } }

export function AdminGroupsPage() {
  const [groups, setGroups] = useState<GroupListItem[]>([])
  const [allUsers, setAllUsers] = useState<UserResponse[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [showCreateDialog, setShowCreateDialog] = useState(false)

  useEffect(() => {
    let mounted = true

    async function loadData() {
      setLoading(true)
      setLoadError(null)
      try {
        const [groupsData, usersData] = await Promise.all([
          request<GroupListItem[]>('GET', API_ENDPOINTS.adminGroups),
          request<UserResponse[]>('GET', API_ENDPOINTS.adminUsers),
        ])
        if (!mounted) {
          return
        }
        setGroups(groupsData)
        setAllUsers(usersData)
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

    loadData()
    return () => {
      mounted = false
    }
  }, [reloadKey])

  function handleGroupUpdated(updated: GroupListItem) {
    setGroups((current) => current.map((group) => (group.id === updated.id ? updated : group)))
  }

  function handleGroupDeleted(groupId: string) {
    setGroups((current) => current.filter((group) => group.id !== groupId))
  }

  function handleCreated() {
    setShowCreateDialog(false)
    setReloadKey((current) => current + 1)
  }

  function retry() {
    setReloadKey((current) => current + 1)
  }

  return (
    <div className="admin-groups">
      <div className="admin-groups-header">
        <h1 className="admin-page-title">权限组管理</h1>
        <button className="ag-btn-primary" type="button" onClick={() => setShowCreateDialog(true)}>
          + 新建权限组
        </button>
      </div>
      {loading && <div className="admin-empty">加载中...</div>}
      {!loading && loadError && (
        <div className="admin-error">
          <p>{loadError}</p>
          <button className="admin-btn-secondary" type="button" onClick={retry}>
            重试
          </button>
        </div>
      )}
      {!loading && !loadError && groups.length === 0 && <div className="admin-empty">暂无权限组</div>}
      {!loading && !loadError && groups.length > 0 && (
        <div className="ag-groups-list">
          {groups.map((group) => (
            <GroupCard
              key={group.id}
              allUsers={allUsers}
              group={group}
              onDeleted={handleGroupDeleted}
              onUpdated={handleGroupUpdated}
            />
          ))}
        </div>
      )}
      {showCreateDialog &&
        createPortal(
          <CreateGroupDialog
            allUsers={allUsers}
            onClose={() => setShowCreateDialog(false)}
            onCreated={handleCreated}
          />,
          document.body,
        )}
    </div>
  )
}

interface GroupCardProps { group: GroupListItem; allUsers: UserResponse[]; onUpdated: (group: GroupListItem) => void; onDeleted: (groupId: string) => void }

function GroupCard({ group, allUsers, onUpdated, onDeleted }: GroupCardProps) {
  const [mode, setMode] = useState<CardMode>('view')
  const [editName, setEditName] = useState(group.name)
  const [editLeaderId, setEditLeaderId] = useState(group.leader.id)
  const [busy, setBusy] = useState(false)
  const [cardError, setCardError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState(false)
  const [members, setMembers] = useState<GroupMemberResponse[] | null>(null)
  const [membersLoading, setMembersLoading] = useState(false)
  const [membersError, setMembersError] = useState<string | null>(null)

  function startRename() {
    setMode('renaming')
    setEditName(group.name)
    setCardError(null)
  }

  function startChangeLeader() {
    setMode('changing-leader')
    setEditLeaderId(group.leader.id)
    setCardError(null)
  }

  function startDelete() {
    setMode('confirming-delete')
    setCardError(null)
  }

  function cancelAction() {
    setMode('view')
    setCardError(null)
  }

  async function handleRename() {
    const nextName = editName.trim()
    if (nextName === '' || nextName.length > 50) {
      setCardError('组名需为 1 到 50 个字符')
      return
    }
    setBusy(true)
    setCardError(null)
    try {
      await request<GroupResponse>(
        'PATCH',
        API_ENDPOINTS.adminGroup(group.id),
        { name: nextName } satisfies UpdateGroupRequest,
      )
      onUpdated({ ...group, name: nextName })
      setMode('view')
    } catch (error) {
      setCardError(errorMessage(error))
    } finally {
      setBusy(false)
    }
  }

  async function handleChangeLeader() {
    if (editLeaderId === '') {
      setCardError('请选择组长')
      return
    }
    setBusy(true)
    setCardError(null)
    try {
      await request<GroupResponse>(
        'PATCH',
        API_ENDPOINTS.adminGroup(group.id),
        { leader_id: editLeaderId } satisfies UpdateGroupRequest,
      )
      const nextLeader = findUser(allUsers, editLeaderId)
      const leaderWasMember = hasMember(members, editLeaderId)
      const nextMembers =
        members !== null && !leaderWasMember
          ? [...members, toLocalMember(group.id, editLeaderId)]
          : members

      if (nextMembers !== members) {
        setMembers(nextMembers)
      }

      onUpdated({
        ...group,
        leader: {
          id: editLeaderId,
          display_name: nextLeader?.display_name ?? editLeaderId.slice(0, 8),
          email: nextLeader?.email ?? '',
        },
        member_count: nextMembers ? nextMembers.length : group.member_count,
      })
      setMode('view')
    } catch (error) {
      setCardError(errorMessage(error))
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete() {
    setBusy(true)
    setCardError(null)
    try {
      await request<null>('DELETE', API_ENDPOINTS.adminGroup(group.id))
      onDeleted(group.id)
    } catch (error) {
      setCardError(errorMessage(error))
      setMode('view')
    } finally {
      setBusy(false)
    }
  }

  async function handleToggleMembers() {
    if (expanded) {
      setExpanded(false)
      return
    }
    setExpanded(true)
    if (members !== null) {
      return
    }
    setMembersLoading(true)
    setMembersError(null)
    try {
      const data = await request<GroupMemberResponse[]>('GET', API_ENDPOINTS.adminGroupMembers(group.id))
      setMembers(data)
      onUpdated({ ...group, member_count: data.length })
    } catch (error) {
      setMembersError(errorMessage(error))
    } finally {
      setMembersLoading(false)
    }
  }

  function handleMemberAdded(member: GroupMemberResponse) {
    setMembers((current) => (current ? [...current, member] : [member]))
    onUpdated({ ...group, member_count: group.member_count + 1 })
  }

  function handleMemberRemoved(userId: string) {
    setMembers((current) => current?.filter((member) => member.user_id !== userId) ?? current)
    onUpdated({ ...group, member_count: group.member_count - 1 })
  }

  return (
    <div className="ag-card">
      <GroupCardHeader
        busy={busy}
        editName={editName}
        group={group}
        mode={mode}
        onCancel={cancelAction}
        onEditNameChange={setEditName}
        onRename={handleRename}
      />
      <GroupCardActions
        allUsers={allUsers}
        busy={busy}
        editLeaderId={editLeaderId}
        groupName={group.name}
        mode={mode}
        onCancel={cancelAction}
        onChangeLeader={handleChangeLeader}
        onDelete={handleDelete}
        onEditLeaderIdChange={setEditLeaderId}
        onStartChangeLeader={startChangeLeader}
        onStartDelete={startDelete}
        onStartRename={startRename}
      />
      {cardError && <div className="ag-card-error">{cardError}</div>}
      <MembersSection
        allUsers={allUsers}
        expanded={expanded}
        groupId={group.id}
        leaderId={group.leader.id}
        memberCount={group.member_count}
        members={members}
        membersError={membersError}
        membersLoading={membersLoading}
        onMemberAdded={handleMemberAdded}
        onMemberRemoved={handleMemberRemoved}
        onToggle={handleToggleMembers}
      />
    </div>
  )
}

interface GroupCardHeaderProps { group: GroupListItem; mode: CardMode; editName: string; busy: boolean; onEditNameChange: (value: string) => void; onRename: () => void; onCancel: () => void }

function GroupCardHeader({
  group,
  mode,
  editName,
  busy,
  onEditNameChange,
  onRename,
  onCancel,
}: GroupCardHeaderProps) {
  return (
    <div className="ag-card-header">
      <div className="ag-card-icon" style={{ background: avatarColor(group.id) }}>
        {firstChar(group.name)}
      </div>
      {mode === 'renaming' ? (
        <div className="ag-inline-edit">
          <input
            autoFocus
            className="ag-inline-input"
            maxLength={50}
            onChange={(event) => onEditNameChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                onRename()
              }
            }}
            value={editName}
          />
          <button
            className="admin-btn-secondary admin-btn-sm"
            disabled={busy}
            onClick={onRename}
            type="button"
          >
            {busy ? '保存中...' : '确认'}
          </button>
          <button className="admin-btn-secondary admin-btn-sm" onClick={onCancel} type="button">
            取消
          </button>
        </div>
      ) : (
        <span className="ag-card-name">{group.name}</span>
      )}
      <div className="ag-card-meta">
        <span>
          组长: <strong>{group.leader.display_name}</strong>
        </span>
        <span>成员: {group.member_count}</span>
        <span>{formatDate(group.created_at)}</span>
      </div>
    </div>
  )
}

interface GroupCardActionsProps { groupName: string; allUsers: UserResponse[]; mode: CardMode; editLeaderId: string; busy: boolean; onStartRename: () => void; onStartChangeLeader: () => void; onStartDelete: () => void; onEditLeaderIdChange: (value: string) => void; onChangeLeader: () => void; onDelete: () => void; onCancel: () => void }

function GroupCardActions({
  groupName,
  allUsers,
  mode,
  editLeaderId,
  busy,
  onStartRename,
  onStartChangeLeader,
  onStartDelete,
  onEditLeaderIdChange,
  onChangeLeader,
  onDelete,
  onCancel,
}: GroupCardActionsProps) {
  return (
    <div className="ag-card-actions">
      {mode === 'view' && (
        <>
          <button className="admin-btn-secondary admin-btn-sm" onClick={onStartRename} type="button">
            重命名
          </button>
          <button
            className="admin-btn-secondary admin-btn-sm"
            onClick={onStartChangeLeader}
            type="button"
          >
            换组长
          </button>
          <button className="admin-btn-danger admin-btn-sm" onClick={onStartDelete} type="button">
            删除
          </button>
        </>
      )}
      {mode === 'changing-leader' && (
        <div className="ag-inline-edit">
          <select
            className="ag-inline-select"
            onChange={(event) => onEditLeaderIdChange(event.target.value)}
            value={editLeaderId}
          >
            <option value="" disabled>
              选择新组长
            </option>
            {allUsers.map((user) => (
              <option key={user.id} value={user.id}>
                {user.display_name} ({user.email})
              </option>
            ))}
          </select>
          <button
            className="admin-btn-secondary admin-btn-sm"
            disabled={busy || editLeaderId === ''}
            onClick={onChangeLeader}
            type="button"
          >
            {busy ? '保存中...' : '确认'}
          </button>
          <button className="admin-btn-secondary admin-btn-sm" onClick={onCancel} type="button">
            取消
          </button>
        </div>
      )}
      {mode === 'confirming-delete' && (
        <div className="ag-inline-edit">
          <span className="ag-delete-warn">确定删除「{groupName}」？</span>
          <button
            className="admin-btn-danger admin-btn-sm"
            disabled={busy}
            onClick={onDelete}
            type="button"
          >
            {busy ? '删除中...' : '确认删除'}
          </button>
          <button className="admin-btn-secondary admin-btn-sm" onClick={onCancel} type="button">
            取消
          </button>
        </div>
      )}
    </div>
  )
}

interface MembersSectionProps { groupId: string; leaderId: string; memberCount: number; allUsers: UserResponse[]; members: GroupMemberResponse[] | null; expanded: boolean; membersLoading: boolean; membersError: string | null; onToggle: () => void; onMemberAdded: (member: GroupMemberResponse) => void; onMemberRemoved: (userId: string) => void }

function MembersSection({
  groupId,
  leaderId,
  memberCount,
  allUsers,
  members,
  expanded,
  membersLoading,
  membersError,
  onToggle,
  onMemberAdded,
  onMemberRemoved,
}: MembersSectionProps) {
  const [adding, setAdding] = useState(false)
  const [addUserId, setAddUserId] = useState('')
  const [addBusy, setAddBusy] = useState(false)
  const [addError, setAddError] = useState<string | null>(null)
  const [removeBusy, setRemoveBusy] = useState<Set<string>>(new Set())
  const [removeErrors, setRemoveErrors] = useState<Record<string, string>>({})
  const availableUsers = allUsers.filter((user) => !hasMember(members, user.id))

  function startAdding() {
    setAdding(true)
    setAddUserId('')
    setAddError(null)
  }

  function cancelAdding() {
    setAdding(false)
    setAddUserId('')
    setAddError(null)
  }

  async function handleAddMember() {
    if (addUserId === '') {
      setAddError('请选择成员')
      return
    }
    setAddBusy(true)
    setAddError(null)
    try {
      await request<null>(
        'POST',
        API_ENDPOINTS.adminGroupMembers(groupId),
        { user_id: addUserId } satisfies AddMemberRequest,
      )
      onMemberAdded(toLocalMember(groupId, addUserId))
      cancelAdding()
    } catch (error) {
      setAddError(errorMessage(error))
    } finally {
      setAddBusy(false)
    }
  }

  async function handleRemoveMember(userId: string) {
    setRemoveBusy((current) => {
      const next = new Set(current)
      next.add(userId)
      return next
    })
    setRemoveErrors((current) => {
      const next = { ...current }
      delete next[userId]
      return next
    })
    try {
      await request<null>('DELETE', API_ENDPOINTS.adminGroupMember(groupId, userId))
      onMemberRemoved(userId)
    } catch (error) {
      setRemoveErrors((current) => ({ ...current, [userId]: errorMessage(error) }))
    } finally {
      setRemoveBusy((current) => {
        const next = new Set(current)
        next.delete(userId)
        return next
      })
    }
  }

  return (
    <div className="ag-members-section">
      <button className="ag-members-toggle" onClick={onToggle} type="button">
        {expanded ? '▼' : '▶'} 成员列表 ({memberCount})
      </button>
      {expanded && (
        <div className="ag-members-content">
          {membersLoading && <div className="ag-members-hint">加载中...</div>}
          {!membersLoading && membersError && (
            <div className="ag-members-hint ag-members-hint--error">{membersError}</div>
          )}
          {!membersLoading && !membersError && members !== null && (
            <>
              <MemberList
                allUsers={allUsers}
                leaderId={leaderId}
                members={members}
                onRemove={handleRemoveMember}
                removeBusy={removeBusy}
                removeErrors={removeErrors}
              />
              {!adding && availableUsers.length > 0 && (
                <button className="ag-add-member-btn" onClick={startAdding} type="button">
                  + 添加成员
                </button>
              )}
              {!adding && availableUsers.length === 0 && (
                <div className="ag-members-hint">暂无可添加成员</div>
              )}
              {adding && (
                <AddMemberEditor
                  addBusy={addBusy}
                  addError={addError}
                  addUserId={addUserId}
                  availableUsers={availableUsers}
                  onAdd={handleAddMember}
                  onCancel={cancelAdding}
                  onUserChange={setAddUserId}
                />
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

interface MemberListProps { members: GroupMemberResponse[]; allUsers: UserResponse[]; leaderId: string; removeBusy: Set<string>; removeErrors: Record<string, string>; onRemove: (userId: string) => void }

function MemberList({
  members,
  allUsers,
  leaderId,
  removeBusy,
  removeErrors,
  onRemove,
}: MemberListProps) {
  if (members.length === 0) {
    return <div className="ag-members-hint">暂无成员</div>
  }
  return (
    <div className="ag-members-list">
      {members.map((member) => (
        <MemberRow
          key={member.user_id}
          busy={removeBusy.has(member.user_id)}
          error={removeErrors[member.user_id] ?? null}
          isLeader={member.user_id === leaderId}
          onRemove={onRemove}
          user={findUser(allUsers, member.user_id)}
          userId={member.user_id}
        />
      ))}
    </div>
  )
}

interface MemberRowProps { userId: string; user: UserResponse | undefined; isLeader: boolean; busy: boolean; error: string | null; onRemove: (userId: string) => void }

function MemberRow({ userId, user, isLeader, busy, error, onRemove }: MemberRowProps) {
  const displayName = user?.display_name ?? userId.slice(0, 8)
  return (
    <div className="ag-member-row">
      <div className="ag-member-avatar" style={{ background: avatarColor(userId) }}>
        {firstChar(displayName)}
      </div>
      <span className="ag-member-name">
        {displayName} {isLeader && <span className="ag-leader-tag">(组长)</span>}
      </span>
      <span className="ag-member-email">{user?.email ?? '—'}</span>
      <div className="ag-member-actions">
        {!isLeader && (
          <button
            className="ag-member-remove"
            disabled={busy}
            onClick={() => onRemove(userId)}
            type="button"
          >
            {busy ? '移除中...' : '移除'}
          </button>
        )}
        {error && <span className="row-feedback row-feedback--error">{error}</span>}
      </div>
    </div>
  )
}

interface AddMemberEditorProps { availableUsers: UserResponse[]; addUserId: string; addBusy: boolean; addError: string | null; onUserChange: (value: string) => void; onAdd: () => void; onCancel: () => void }

function AddMemberEditor({
  availableUsers,
  addUserId,
  addBusy,
  addError,
  onUserChange,
  onAdd,
  onCancel,
}: AddMemberEditorProps) {
  return (
    <div className="ag-add-member-row">
      <select
        className="ag-add-member-select"
        onChange={(event) => onUserChange(event.target.value)}
        value={addUserId}
      >
        <option value="">选择成员</option>
        {availableUsers.map((user) => (
          <option key={user.id} value={user.id}>
            {user.display_name} ({user.email})
          </option>
        ))}
      </select>
      <button
        className="admin-btn-secondary admin-btn-sm"
        disabled={addBusy || addUserId === ''}
        onClick={onAdd}
        type="button"
      >
        {addBusy ? '添加中...' : '添加'}
      </button>
      <button className="admin-btn-secondary admin-btn-sm" onClick={onCancel} type="button">
        取消
      </button>
      {addError && <span className="ag-add-member-feedback row-feedback row-feedback--error">{addError}</span>}
    </div>
  )
}

interface CreateGroupDialogProps { allUsers: UserResponse[]; onClose: () => void; onCreated: () => void }

function CreateGroupDialog({ allUsers, onClose, onCreated }: CreateGroupDialogProps) {
  const [name, setName] = useState('')
  const [leaderId, setLeaderId] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  function handleOverlayClick(event: ReactMouseEvent<HTMLDivElement>) {
    if (event.target === event.currentTarget) {
      onClose()
    }
  }

  async function handleCreate() {
    const nextName = name.trim()
    if (nextName === '' || nextName.length > 50) {
      setSaveError('组名需为 1 到 50 个字符')
      return
    }
    if (leaderId === '') {
      setSaveError('请选择组长')
      return
    }
    setSaving(true)
    setSaveError(null)
    try {
      await request<GroupResponse>(
        'POST',
        API_ENDPOINTS.adminGroups,
        { name: nextName, leader_id: leaderId } satisfies CreateGroupRequest,
      )
      onCreated()
    } catch (error) {
      setSaveError(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="ag-dialog-overlay" onClick={handleOverlayClick}>
      <div aria-modal="true" className="ag-dialog" onClick={(event) => event.stopPropagation()} role="dialog">
        <div className="ag-dialog-header">
          <h2 className="ag-dialog-title">新建权限组</h2>
          <button className="ag-dialog-close" onClick={onClose} type="button">
            ×
          </button>
        </div>
        <div className="ag-dialog-body">
          <div className="ag-form-field">
            <label className="ag-form-label">组名</label>
            <input
              autoFocus
              className="ag-form-input"
              maxLength={50}
              onChange={(event) => setName(event.target.value)}
              placeholder="输入组名"
              value={name}
            />
          </div>
          <div className="ag-form-field">
            <label className="ag-form-label">组长</label>
            <select
              className="ag-form-select"
              onChange={(event) => setLeaderId(event.target.value)}
              value={leaderId}
            >
              <option value="" disabled>
                选择组长
              </option>
              {allUsers.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.display_name} ({user.email})
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="ag-dialog-footer">
          {saveError && <span className="ag-dialog-error">{saveError}</span>}
          <button className="admin-btn-secondary" onClick={onClose} type="button">
            取消
          </button>
          <button
            className="ag-btn-primary"
            disabled={saving || name.trim() === '' || leaderId === ''}
            onClick={handleCreate}
            type="button"
          >
            {saving ? '创建中...' : '创建'}
          </button>
        </div>
      </div>
    </div>
  )
}
