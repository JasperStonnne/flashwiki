import { useEffect, useRef, type MouseEvent } from 'react'

import type { NodeListItem, PermissionLevel } from '../api/types'
import './context-menu.css'

interface NodeContextMenuProps {
  x: number
  y: number
  node: NodeListItem | null
  isManager: boolean
  onClose: () => void
  onCreateFolder: () => void
  onCreateDoc: () => void
  onRename: () => void
  onDelete: () => void
}

interface MenuItem {
  label: string
  danger?: boolean
  onSelect: () => void
}

const MENU_WIDTH = 160
const MENU_ITEM_HEIGHT = 36
const MENU_MARGIN = 8

function canEdit(perm: PermissionLevel): boolean {
  return perm === 'edit' || perm === 'manage'
}

function canManage(perm: PermissionLevel): boolean {
  return perm === 'manage'
}

function menuPosition(x: number, y: number, itemCount: number) {
  const estimatedHeight = itemCount * MENU_ITEM_HEIGHT + 8
  const maxLeft = window.innerWidth - MENU_WIDTH - MENU_MARGIN
  const maxTop = window.innerHeight - estimatedHeight - MENU_MARGIN

  return {
    left: Math.max(MENU_MARGIN, Math.min(x, maxLeft)),
    top: Math.max(MENU_MARGIN, Math.min(y, maxTop)),
  }
}

export function NodeContextMenu({
  x,
  y,
  node,
  isManager,
  onClose,
  onCreateDoc,
  onCreateFolder,
  onDelete,
  onRename,
}: NodeContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)
  const items = buildMenuItems(node, isManager, onCreateFolder, onCreateDoc, onRename, onDelete)

  useEffect(() => {
    function handleMouseDown(event: globalThis.MouseEvent) {
      if (!menuRef.current?.contains(event.target as Node)) {
        onClose()
      }
    }

    function handleKeyDown(event: globalThis.KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleMouseDown)
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      document.removeEventListener('mousedown', handleMouseDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [onClose])

  if (items.length === 0) {
    return null
  }

  const position = menuPosition(x, y, items.length)

  function handleItemClick(event: MouseEvent<HTMLButtonElement>, item: MenuItem) {
    event.stopPropagation()
    onClose()
    item.onSelect()
  }

  return (
    <div
      className="context-menu"
      ref={menuRef}
      style={{ left: position.left, top: position.top }}
    >
      {items.map((item) => (
        <button
          className={`context-menu-item${item.danger ? ' context-menu-item--danger' : ''}`}
          key={item.label}
          onClick={(event) => handleItemClick(event, item)}
          type="button"
        >
          <span className="context-menu-icon" aria-hidden="true" />
          <span>{item.label}</span>
        </button>
      ))}
    </div>
  )
}

function buildMenuItems(
  node: NodeListItem | null,
  isManager: boolean,
  onCreateFolder: () => void,
  onCreateDoc: () => void,
  onRename: () => void,
  onDelete: () => void,
): MenuItem[] {
  if (node === null) {
    return isManager
      ? [
          { label: '新建根文件夹', onSelect: onCreateFolder },
          { label: '新建根文档', onSelect: onCreateDoc },
        ]
      : []
  }

  const items: MenuItem[] = []
  if (node.kind === 'folder' && canEdit(node.permission)) {
    items.push({ label: '新建文件夹', onSelect: onCreateFolder })
    items.push({ label: '新建文档', onSelect: onCreateDoc })
  }
  if (canEdit(node.permission)) {
    items.push({ label: '重命名', onSelect: onRename })
  }
  if (canManage(node.permission)) {
    items.push({ danger: true, label: '删除', onSelect: onDelete })
  }

  return items
}
