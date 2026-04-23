import { useEffect, useState, type Dispatch, type KeyboardEvent, type SetStateAction } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { request } from '../api/client'
import { API_ENDPOINTS } from '../api/endpoints'
import type { NodeListItem } from '../api/types'
import './node-tree.css'

const EXPANDED_NODES_KEY = 'fpgwiki_expanded_nodes'

interface NodeTreeProps {
  parentId: string | null
  depth: number
}

interface NodeTreeBranchProps extends NodeTreeProps {
  activeDocId: string | null
  expandedIds: Set<string>
  onDocumentClick: (nodeId: string) => void
  setExpandedIds: Dispatch<SetStateAction<Set<string>>>
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

function useNodeBranchData(parentId: string | null) {
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
  }, [parentId])

  return { failed, loading, nodes }
}

export function NodeTree({ parentId, depth }: NodeTreeProps) {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => readExpandedNodeIds())
  const navigate = useNavigate()
  const { id: activeDocId } = useParams<{ id: string }>()

  useEffect(() => {
    writeExpandedNodeIds(expandedIds)
  }, [expandedIds])

  function handleDocumentClick(nodeId: string) {
    navigate(`/doc/${nodeId}`)
  }

  return (
    <NodeTreeBranch
      activeDocId={activeDocId ?? null}
      depth={depth}
      expandedIds={expandedIds}
      onDocumentClick={handleDocumentClick}
      parentId={parentId}
      setExpandedIds={setExpandedIds}
    />
  )
}

function NodeTreeBranch({
  activeDocId,
  depth,
  expandedIds,
  onDocumentClick,
  parentId,
  setExpandedIds,
}: NodeTreeBranchProps) {
  const { failed, loading, nodes } = useNodeBranchData(parentId)

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

  if (parentId === null && nodes.length === 0) {
    return <div className="tree-empty">暂无文档</div>
  }

  return (
    <div className="tree-group">
      {nodes.map((node) => (
        <TreeNodeRow
          activeDocId={activeDocId}
          depth={depth}
          expandedIds={expandedIds}
          key={node.id}
          node={node}
          onDocumentClick={onDocumentClick}
          onNodeClick={handleNodeClick}
          onToggleFolder={toggleFolder}
          setExpandedIds={setExpandedIds}
        />
      ))}
    </div>
  )
}

interface TreeNodeRowProps {
  activeDocId: string | null
  depth: number
  expandedIds: Set<string>
  node: NodeListItem
  onDocumentClick: (nodeId: string) => void
  onNodeClick: (node: NodeListItem) => void
  onToggleFolder: (nodeId: string) => void
  setExpandedIds: Dispatch<SetStateAction<Set<string>>>
}

function TreeNodeRow({
  activeDocId,
  depth,
  expandedIds,
  node,
  onDocumentClick,
  onNodeClick,
  onToggleFolder,
  setExpandedIds,
}: TreeNodeRowProps) {
  const active = node.kind === 'doc' && activeDocId === node.id
  const expanded = expandedIds.has(node.id)

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return
    }

    event.preventDefault()
    onNodeClick(node)
  }

  return (
    <div className="tree-item">
      <div
        aria-expanded={node.kind === 'folder' ? expanded : undefined}
        className={`tree-node${active ? ' tree-node--active' : ''}`}
        onClick={() => onNodeClick(node)}
        onKeyDown={handleKeyDown}
        role="button"
        style={{ paddingLeft: `${12 + depth * 20}px` }}
        tabIndex={0}
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
        <span className="tree-label">{node.title}</span>
      </div>
      {node.kind === 'folder' && expanded && (
        <NodeTreeBranch
          activeDocId={activeDocId}
          depth={depth + 1}
          expandedIds={expandedIds}
          onDocumentClick={onDocumentClick}
          parentId={node.id}
          setExpandedIds={setExpandedIds}
        />
      )}
    </div>
  )
}
