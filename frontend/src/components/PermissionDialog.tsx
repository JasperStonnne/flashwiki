import { useEffect, useState, type CSSProperties } from 'react'
import { createPortal } from 'react-dom'

import { request } from '../api/client'
import { API_ENDPOINTS } from '../api/endpoints'
import type {
  NodePermissionResult,
  PermissionLevel,
  SetNodePermissionsRequest,
  SubjectType,
} from '../api/types'
import './permission-dialog.css'

interface PermissionDialogProps {
  nodeId: string
  nodeTitle: string
  onClose: () => void
}

type LocalPermissionEntry = {
  key: string
  subjectType: SubjectType
  subjectId: string
  level: PermissionLevel
}

type NewEntryState = {
  subjectType: SubjectType
  subjectId: string
  level: PermissionLevel
}

function newEntryState(): NewEntryState {
  return {
    subjectType: 'user',
    subjectId: '',
    level: 'readable',
  }
}

function createLocalEntry(
  subjectType: SubjectType,
  subjectId: string,
  level: PermissionLevel,
): LocalPermissionEntry {
  return {
    key: crypto.randomUUID(),
    level,
    subjectId,
    subjectType,
  }
}

function shortSubjectID(subjectId: string): string {
  return subjectId.slice(0, 8)
}

function subjectLabel(subjectType: SubjectType): string {
  return subjectType === 'user' ? '用户' : '组'
}

function levelSelectStyle(level: PermissionLevel): CSSProperties {
  const colorMap: Record<PermissionLevel, string> = {
    manage: 'var(--primary)',
    edit: 'var(--green)',
    readable: 'var(--orange)',
    none: 'var(--gray-300)',
  }

  return { borderColor: colorMap[level] }
}

function toLocalEntries(result: NodePermissionResult): LocalPermissionEntry[] {
  return result.permissions.map((entry) =>
    createLocalEntry(entry.subject_type, entry.subject_id, entry.level),
  )
}

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message
  }

  return '保存失败'
}

export function PermissionDialog({ nodeId, nodeTitle, onClose }: PermissionDialogProps) {
  const [adding, setAdding] = useState(false)
  const [entries, setEntries] = useState<LocalPermissionEntry[]>([])
  const [inheritedFrom, setInheritedFrom] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [newEntry, setNewEntry] = useState<NewEntryState>(() => newEntryState())
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let mounted = true

    async function loadPermissions() {
      setLoading(true)
      setLoadError(null)
      setSaveError(null)

      try {
        const result = await request<NodePermissionResult>('GET', API_ENDPOINTS.nodePermissions(nodeId))
        if (!mounted) {
          return
        }

        setEntries(toLocalEntries(result))
        setInheritedFrom(result.inherited_from)
      } catch (error) {
        if (!mounted) {
          return
        }

        setEntries([])
        setInheritedFrom(null)
        setLoadError(errorMessage(error))
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }

    loadPermissions()

    return () => {
      mounted = false
    }
  }, [nodeId])

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [onClose])

  function handleOverlayClick(event: React.MouseEvent<HTMLDivElement>) {
    if (event.target === event.currentTarget) {
      onClose()
    }
  }

  function handleLevelChange(entryKey: string, level: PermissionLevel) {
    setEntries((current) =>
      current.map((entry) => (entry.key === entryKey ? { ...entry, level } : entry)),
    )
    setSaveError(null)
  }

  function handleRemoveEntry(entryKey: string) {
    setEntries((current) => current.filter((entry) => entry.key !== entryKey))
    setSaveError(null)
  }

  function handleAddStart() {
    setAdding(true)
    setNewEntry(newEntryState())
    setSaveError(null)
  }

  function handleAddCancel() {
    setAdding(false)
    setNewEntry(newEntryState())
  }

  function handleAddConfirm() {
    const subjectId = newEntry.subjectId.trim()
    if (subjectId === '') {
      setSaveError('subject_id 不能为空')
      return
    }

    const duplicated = entries.some(
      (entry) =>
        entry.subjectType === newEntry.subjectType &&
        entry.subjectId.toLowerCase() === subjectId.toLowerCase(),
    )
    if (duplicated) {
      setSaveError('权限条目不能重复')
      return
    }

    setEntries((current) => [
      ...current,
      createLocalEntry(newEntry.subjectType, subjectId, newEntry.level),
    ])
    setAdding(false)
    setNewEntry(newEntryState())
    setSaveError(null)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError(null)

    try {
      await request<null>('PUT', API_ENDPOINTS.nodePermissions(nodeId), {
        permissions: entries.map((entry) => ({
          subject_type: entry.subjectType,
          subject_id: entry.subjectId,
          level: entry.level,
        })),
      } satisfies SetNodePermissionsRequest)
      onClose()
    } catch (error) {
      setSaveError(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  const dialog = (
    <div className="dialog-overlay" onClick={handleOverlayClick}>
      <div
        className="dialog"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="permission-dialog-title"
      >
        <div className="dialog-header">
          <h2 className="dialog-title" id="permission-dialog-title">
            权限设置 — {nodeTitle}
          </h2>
          <button className="dialog-close" onClick={onClose} type="button">
            ×
          </button>
        </div>
        <div className="dialog-body">
          {inheritedFrom && <div className="perm-inherited-banner">当前无自有权限行，继承自上级节点</div>}
          {loading ? (
            <div className="perm-loading">加载中...</div>
          ) : loadError ? (
            <div className="perm-empty">{loadError}</div>
          ) : (
            <>
              {entries.length === 0 ? (
                <div className="perm-empty">暂无权限条目</div>
              ) : (
                entries.map((entry) => (
                  <div className="perm-row" key={entry.key}>
                    <span className="perm-subject-tag">{subjectLabel(entry.subjectType)}</span>
                    <span className="perm-subject-id">{shortSubjectID(entry.subjectId)}</span>
                    <select
                      className="perm-level-select"
                      onChange={(event) =>
                        handleLevelChange(entry.key, event.target.value as PermissionLevel)
                      }
                      style={levelSelectStyle(entry.level)}
                      value={entry.level}
                    >
                      <option value="manage">manage</option>
                      <option value="edit">edit</option>
                      <option value="readable">readable</option>
                      <option value="none">none</option>
                    </select>
                    <button
                      className="perm-remove-btn"
                      onClick={() => handleRemoveEntry(entry.key)}
                      type="button"
                    >
                      ×
                    </button>
                  </div>
                ))
              )}
              {adding ? (
                <div className="perm-add-row">
                  <select
                    onChange={(event) =>
                      setNewEntry((current) => ({
                        ...current,
                        subjectType: event.target.value as SubjectType,
                      }))
                    }
                    value={newEntry.subjectType}
                  >
                    <option value="user">user</option>
                    <option value="group">group</option>
                  </select>
                  <input
                    onChange={(event) =>
                      setNewEntry((current) => ({ ...current, subjectId: event.target.value }))
                    }
                    placeholder="输入 UUID"
                    value={newEntry.subjectId}
                  />
                  <select
                    onChange={(event) =>
                      setNewEntry((current) => ({
                        ...current,
                        level: event.target.value as PermissionLevel,
                      }))
                    }
                    style={levelSelectStyle(newEntry.level)}
                    value={newEntry.level}
                  >
                    <option value="manage">manage</option>
                    <option value="edit">edit</option>
                    <option value="readable">readable</option>
                    <option value="none">none</option>
                  </select>
                  <button className="btn-secondary" onClick={handleAddConfirm} type="button">
                    确认
                  </button>
                  <button className="btn-secondary" onClick={handleAddCancel} type="button">
                    取消
                  </button>
                </div>
              ) : (
                <button className="perm-add-btn" onClick={handleAddStart} type="button">
                  + 添加权限
                </button>
              )}
            </>
          )}
        </div>
        <div className="dialog-footer">
          {saveError && <span className="dialog-error">{saveError}</span>}
          <button className="btn-secondary" onClick={onClose} type="button">
            取消
          </button>
          <button
            className="btn-primary"
            disabled={saving || loading || loadError !== null}
            onClick={handleSave}
            type="button"
          >
            {saving ? '保存中...' : '保存'}
          </button>
        </div>
      </div>
    </div>
  )

  return createPortal(dialog, document.body)
}
