import { NodeTree } from './NodeTree'
import './node-tree.css'

export function Sidebar() {
  return (
    <nav className="sidebar-nav">
      <div className="sidebar-section-label">WORKSPACE</div>
      <NodeTree parentId={null} depth={0} />
    </nav>
  )
}
