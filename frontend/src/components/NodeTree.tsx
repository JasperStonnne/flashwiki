import {
  useEffect,
  useState,
  type Dispatch,
  type KeyboardEvent,
  type MouseEvent as ReactMouseEvent,
  type SetStateAction,
} from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'
import { request } from '../api/client'
import { API_ENDPOINTS } from '../api/endpoints'
import type {
  CreateNodeRequest,
  NodeKind,
  NodeListItem,
  NodeResponse,
  UpdateNodeRequest,
} from '../api/types'
import { InlineNodeInput } from './InlineNodeInput'
import { NodeContextMenu } from './NodeContextMenu'
import { PermissionDialog } from './PermissionDialog'
import './node-tree.css'

const EXPANDED_NODES_KEY = 'fpgwiki_expanded_nodes'
const ROOT_BRANCH_KEY = 'root'
const DEFAULT_FOLDER_TITLE = '未命名文件夹'
const DEFAULT_DOC_TITLE = '未命名文档'

interface NodeTreeProps {
  parentId: string | null
  depth: number
}

interface CreateDraft {
  parentId: string | null
  kind: NodeKind
}

interface ContextMenuState {
  x: number
  y: number
  node: NodeListItem | null
}

interface PermissionTarget {
  nodeId: string
  nodeTitle: string
}

interface NodeTreeBranchProps extends NodeTreeProps {
  activeDocId: string | null
  creatingNode: CreateDraft | null
  expandedIds: Set<string>
  getRefreshKey: (parentId: string | null) => number
  onCancelCreate: () => void
  onCancelRename: () => void
  onContextMenu: (node: NodeListItem, x: number, y: number) => void
  onCreateConfirm: (parentId: string | null, kind: NodeKind, title: string) => void
  onDocumentClick: (nodeId: string) => void
  onRenameConfirm: (node: NodeListItem, title: string) => void
  renamingNodeId: string | null
  setExpandedIds: Dispatch<SetStateAction<Set<string>>>
}

interface TreeNodeRowProps {
  active: boolean
  depth: number
  expanded: boolean
  isRenaming: boolean
  node: NodeListItem
  onCancelRename: () => void
  onContextMenu: (node: NodeListItem, x: number, y: number) => void
  onNodeClick: (node: NodeListItem) => void
  onRenameConfirm: (node: NodeListItem, title: string) => void
  onToggleFolder: (nodeId: string) => void
}

function branchKey(parentId: string | null): string {
  return parentId ?? ROOT_BRANCH_KEY
}

function readExpandedNodeIds(): Set<string> {
  const raw = localStorage.getItem(EXPANDED_NODES_KEY)
  if (!raw) {
    return new Set()
  }

  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) {
      return new Set()
    }

    return new Set(parsed.filter((value): value is string => typeof value === 'string'))
  } catch {
    return new Set()
  }
}

function writeExpandedNodeIds(expandedIds: Set<string>) {
  try {
    localStorage.setItem(EXPANDED_NODES_KEY, JSON.stringify([...expandedIds]))
  } catch {
    // Persisting tree UI state is best-effort and should not block navigation.
  }
}

function nodeChildrenPath(parentId: string | null): string {
  if (parentId === null) {
    return API_ENDPOINTS.nodes
  }

  return `${API_ENDPOINTS.nodes}?parent=${parentId}`
}

function defaultTitleFor(kind: NodeKind): string {
  return kind === 'folder' ? DEFAULT_FOLDER_TITLE : DEFAULT_DOC_TITLE
}

function canOpenNodeMenu(node: NodeListItem): boolean {
  return node.permission === 'edit' || node.permission === 'manage'
}

function useNodeBranchData(parentId: string | null, refreshKey: number) {
  const [nodes, setNodes] = useState<NodeListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let mounted = true

    async function loadNodes() {
      setLoading(true)
      setFailed(false)

      try {
        const nextNodes = await request<NodeListItem[]>('GET', nodeChildrenPath(parentId))
        if (mounted) {
          setNodes(nextNodes)
        }
      } catch {
        if (mounted) {
          setNodes([])
          setFailed(true)
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }

    loadNodes()

    return () => {
      mounted = false
    }
  }, [parentId, refreshKey])

  return { failed, loading, nodes }
}

export function NodeTree({ parentId, depth }: NodeTreeProps) {
  const { userRole } = useAuth()
  const navigate = useNavigate()
  const { id: activeDocId } = useParams<{ id: string }>()
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [creatingNode, setCreatingNode] = useState<CreateDraft | null>(null)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => readExpandedNodeIds())
  const [permissionTarget, setPermissionTarget] = useState<PermissionTarget | null>(null)
  const [refreshMap, setRefreshMap] = useState<Record<string, number>>({})
  const [renamingNodeId, setRenamingNodeId] = useState<string | null>(null)
  const isManager = userRole === 'manager'

  useEffect(() => {
    writeExpandedNodeIds(expandedIds)
  }, [expandedIds])

  function getRefreshKey(targetParentId: string | null): number {
    return refreshMap[branchKey(targetParentId)] ?? 0
  }

  function refreshBranch(targetParentId: string | null) {
    const key = branchKey(targetParentId)
    setRefreshMap((current) => ({
      ...current,
      [key]: (current[key] ?? 0) + 1,
    }))
  }

  function ensureExpanded(nodeId: string) {
    setExpandedIds((current) => {
      if (current.has(nodeId)) {
        return current
      }

      const next = new Set(current)
      next.add(nodeId)
      return next
    })
  }

  function closeContextMenu() {
    setContextMenu(null)
  }

  function handleDocumentClick(nodeId: string) {
    navigate(`/doc/${nodeId}`)
  }

  function handleNodeContextMenu(node: NodeListItem, x: number, y: number) {
    if (!canOpenNodeMenu(node)) {
      closeContextMenu()
      return
    }

    setContextMenu({ node, x, y })
  }

  function handleSurfaceContextMenu(event: ReactMouseEvent<HTMLDivElement>) {
    const target = event.target as HTMLElement
    if (
      target.closest('.context-menu') ||
      target.closest('.inline-node-input') ||
      target.closest('.tree-node')
    ) {
      return
    }

    event.preventDefault()
    if (!isManager) {
      closeContextMenu()
      return
    }

    setContextMenu({ node: null, x: event.clientX, y: event.clientY })
  }

  function beginCreate(parentForCreate: string | null, kind: NodeKind) {
    closeContextMenu()
    setRenamingNodeId(null)
    setCreatingNode({ parentId: parentForCreate, kind })
    if (parentForCreate !== null) {
      ensureExpanded(parentForCreate)
    }
  }

  function beginRename(nodeId: string) {
    closeContextMenu()
    setCreatingNode(null)
    setRenamingNodeId(nodeId)
  }

  function cancelCreate() {
    setCreatingNode(null)
  }

  function cancelRename() {
    setRenamingNodeId(null)
  }

  async function handleCreateConfirm(parentForCreate: string | null, kind: NodeKind, title: string) {
    setCreatingNode(null)

    try {
      await request<NodeResponse>('POST', API_ENDPOINTS.nodes, {
        parent_id: parentForCreate,
        kind,
        title,
      } satisfies CreateNodeRequest)
      refreshBranch(parentForCreate)
    } catch {
      // CRUD errors are intentionally silent in the MVP tree interactions.
    }
  }

  async function handleRenameConfirm(node: NodeListItem, title: string) {
    setRenamingNodeId(null)

    try {
      await request<NodeResponse>('PATCH', API_ENDPOINTS.node(node.id), {
        title,
      } satisfies UpdateNodeRequest)
      refreshBranch(node.parent_id)
    } catch {
      // CRUD errors are intentionally silent in the MVP tree interactions.
    }
  }

  async function handleDeleteNode(node: NodeListItem) {
    closeContextMenu()
    setCreatingNode(null)
    setRenamingNodeId((current) => (current === node.id ? null : current))
    setExpandedIds((current) => {
      if (!current.has(node.id)) {
        return current
      }

      const next = new Set(current)
      next.delete(node.id)
      return next
    })

    try {
      await request<null>('DELETE', API_ENDPOINTS.node(node.id))
      refreshBranch(node.parent_id)
      if (activeDocId === node.id) {
        navigate('/')
      }
    } catch {
      // CRUD errors are intentionally silent in the MVP tree interactions.
    }
  }

  function handleCreateFolder() {
    if (contextMenu?.node?.kind === 'folder') {
      beginCreate(contextMenu.node.id, 'folder')
      return
    }

    if (contextMenu?.node === null || contextMenu === null) {
      beginCreate(null, 'folder')
    }
  }

  function handleCreateDoc() {
    if (contextMenu?.node?.kind === 'folder') {
      beginCreate(contextMenu.node.id, 'doc')
      return
    }

    if (contextMenu?.node === null || contextMenu === null) {
      beginCreate(null, 'doc')
    }
  }

  function handleRenameFromMenu() {
    if (!contextMenu?.node) {
      return
    }

    beginRename(contextMenu.node.id)
  }

  function handleDeleteFromMenu() {
    if (!contextMenu?.node) {
      return
    }

    void handleDeleteNode(contextMenu.node)
  }

  function handleSetPermissions() {
    if (!contextMenu?.node) {
      return
    }

    setPermissionTarget({
      nodeId: contextMenu.node.id,
      nodeTitle: contextMenu.node.title,
    })
    closeContextMenu()
  }

  return (
    <div
      className="tree-surface"
      onContextMenu={handleSurfaceContextMenu}
      style={{ display: 'flex', flexDirection: 'column', minHeight: '100%' }}
    >
      <NodeTreeBranch
        activeDocId={activeDocId ?? null}
        creatingNode={creatingNode}
        depth={depth}
        expandedIds={expandedIds}
        getRefreshKey={getRefreshKey}
        onCancelCreate={cancelCreate}
        onCancelRename={cancelRename}
        onContextMenu={handleNodeContextMenu}
        onCreateConfirm={handleCreateConfirm}
        onDocumentClick={handleDocumentClick}
        onRenameConfirm={handleRenameConfirm}
        parentId={parentId}
        renamingNodeId={renamingNodeId}
        setExpandedIds={setExpandedIds}
      />
      <div style={{ flex: 1 }} />
      {contextMenu && (
        <NodeContextMenu
          isManager={isManager}
          node={contextMenu.node}
          onClose={closeContextMenu}
          onCreateDoc={handleCreateDoc}
          onCreateFolder={handleCreateFolder}
          onDelete={handleDeleteFromMenu}
          onRename={handleRenameFromMenu}
          onSetPermissions={handleSetPermissions}
          x={contextMenu.x}
          y={contextMenu.y}
        />
      )}
      {permissionTarget && (
        <PermissionDialog
          nodeId={permissionTarget.nodeId}
          nodeTitle={permissionTarget.nodeTitle}
          onClose={() => setPermissionTarget(null)}
        />
      )}
    </div>
  )
}

function NodeTreeBranch({
  activeDocId,
  creatingNode,
  depth,
  expandedIds,
  getRefreshKey,
  onCancelCreate,
  onCancelRename,
  onContextMenu,
  onCreateConfirm,
  onDocumentClick,
  onRenameConfirm,
  parentId,
  renamingNodeId,
  setExpandedIds,
}: NodeTreeBranchProps) {
  const { failed, loading, nodes } = useNodeBranchData(parentId, getRefreshKey(parentId))
  const creatingHere = creatingNode?.parentId === parentId ? creatingNode : null

  function toggleFolder(nodeId: string) {
    setExpandedIds((current) => {
      const next = new Set(current)
      if (next.has(nodeId)) {
        next.delete(nodeId)
      } else {
        next.add(nodeId)
      }
      return next
    })
  }

  function handleNodeClick(node: NodeListItem) {
    if (renamingNodeId === node.id) {
      return
    }

    if (node.kind === 'folder') {
      toggleFolder(node.id)
      return
    }

    onDocumentClick(node.id)
  }

  if (loading) {
    return <div className="tree-loading">加载中...</div>
  }

  if (failed) {
    return <div className="tree-empty">加载失败</div>
  }

  if (parentId === null && nodes.length === 0 && creatingHere === null) {
    return <div className="tree-empty">暂无文档</div>
  }

  return (
    <div className="tree-group">
      {nodes.map((node) => {
        const expanded = expandedIds.has(node.id)

        return (
          <div className="tree-item" key={node.id}>
            <TreeNodeRow
              active={node.kind === 'doc' && activeDocId === node.id}
              depth={depth}
              expanded={expanded}
              isRenaming={renamingNodeId === node.id}
              node={node}
              onCancelRename={onCancelRename}
              onContextMenu={onContextMenu}
              onNodeClick={handleNodeClick}
              onRenameConfirm={onRenameConfirm}
              onToggleFolder={toggleFolder}
            />
            {node.kind === 'folder' && expanded && (
              <NodeTreeBranch
                activeDocId={activeDocId}
                creatingNode={creatingNode}
                depth={depth + 1}
                expandedIds={expandedIds}
                getRefreshKey={getRefreshKey}
                onCancelCreate={onCancelCreate}
                onCancelRename={onCancelRename}
                onContextMenu={onContextMenu}
                onCreateConfirm={onCreateConfirm}
                onDocumentClick={onDocumentClick}
                onRenameConfirm={onRenameConfirm}
                parentId={node.id}
                renamingNodeId={renamingNodeId}
                setExpandedIds={setExpandedIds}
              />
            )}
          </div>
        )
      })}
      {creatingHere && (
        <CreateNodeRow
          depth={depth}
          kind={creatingHere.kind}
          onCancel={onCancelCreate}
          onConfirm={(title) => onCreateConfirm(parentId, creatingHere.kind, title)}
        />
      )}
    </div>
  )
}

function TreeNodeRow({
  active,
  depth,
  expanded,
  isRenaming,
  node,
  onCancelRename,
  onContextMenu,
  onNodeClick,
  onRenameConfirm,
  onToggleFolder,
}: TreeNodeRowProps) {
  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (isRenaming || (event.key !== 'Enter' && event.key !== ' ')) {
      return
    }

    event.preventDefault()
    onNodeClick(node)
  }

  return (
    <div
      aria-expanded={node.kind === 'folder' ? expanded : undefined}
      className={`tree-node${active ? ' tree-node--active' : ''}`}
      onClick={() => onNodeClick(node)}
      onContextMenu={(event) => {
        event.preventDefault()
        event.stopPropagation()
        onContextMenu(node, event.clientX, event.clientY)
      }}
      onKeyDown={handleKeyDown}
      role="button"
      style={{ paddingLeft: `${12 + depth * 20}px` }}
      tabIndex={isRenaming ? -1 : 0}
    >
      {node.kind === 'folder' ? (
        <span
          className="tree-toggle"
          onClick={(event) => {
            event.stopPropagation()
            onToggleFolder(node.id)
          }}
        >
          {expanded ? '▼' : '▶'}
        </span>
      ) : (
        <span className="tree-toggle tree-toggle--placeholder" aria-hidden="true" />
      )}
      <span className="tree-icon" aria-hidden="true">
        {node.kind === 'folder' ? '📁' : '📄'}
      </span>
      <span className="tree-label" style={{ flex: 1 }}>
        {isRenaming ? (
          <InlineNodeInput
            defaultValue={node.title}
            onCancel={onCancelRename}
            onConfirm={(title) => onRenameConfirm(node, title)}
          />
        ) : (
          node.title
        )}
      </span>
    </div>
  )
}

interface CreateNodeRowProps {
  depth: number
  kind: NodeKind
  onCancel: () => void
  onConfirm: (title: string) => void
}

function CreateNodeRow({ depth, kind, onCancel, onConfirm }: CreateNodeRowProps) {
  return (
    <div className="tree-node" style={{ paddingLeft: `${12 + depth * 20}px` }}>
      <span className="tree-toggle tree-toggle--placeholder" aria-hidden="true" />
      <span className="tree-icon" aria-hidden="true">
        {kind === 'folder' ? '📁' : '📄'}
      </span>
      <span className="tree-label" style={{ flex: 1 }}>
        <InlineNodeInput
          defaultValue={defaultTitleFor(kind)}
          onCancel={onCancel}
          onConfirm={onConfirm}
        />
      </span>
    </div>
  )
}
