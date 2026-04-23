export type PermissionLevel = 'manage' | 'edit' | 'readable' | 'none'
export type NodeKind = 'folder' | 'doc'
export type UserRole = 'manager' | 'member'
export type SubjectType = 'user' | 'group'

export interface NodeResponse {
  id: string
  parent_id: string | null
  kind: NodeKind
  title: string
  owner_id: string
  created_at: string
  updated_at: string
}

export interface NodeDetailResponse extends NodeResponse {
  permission: PermissionLevel
}

export interface NodeListItem extends NodeResponse {
  permission: PermissionLevel
  has_children: boolean
}

export interface CreateNodeRequest {
  parent_id: string | null
  kind: NodeKind
  title: string
}

export interface UpdateNodeRequest {
  title?: string
  parent_id?: string | null
}

export interface NodePermissionEntry {
  id: string
  node_id: string
  subject_type: SubjectType
  subject_id: string
  level: PermissionLevel
  created_at: string
  updated_at: string
}

export interface NodePermissionResult {
  node_id: string
  permissions: NodePermissionEntry[]
  inherited_from: string | null
}

export interface SetPermissionEntry {
  subject_type: SubjectType
  subject_id: string
  level: PermissionLevel
}

export interface SetNodePermissionsRequest {
  permissions: SetPermissionEntry[]
}

export interface UserResponse {
  id: string
  email: string
  display_name: string
  role: UserRole
  locale: string
  created_at: string
}

export interface ChangeRoleRequest {
  role: UserRole
}

export interface GroupLeader {
  id: string
  display_name: string
  email: string
}

export interface GroupListItem {
  id: string
  name: string
  leader: GroupLeader
  member_count: number
  created_at: string
}

export interface GroupResponse {
  id: string
  name: string
  leader_id: string
  created_at: string
  updated_at: string
}

export interface CreateGroupRequest {
  name: string
  leader_id: string
}

export interface UpdateGroupRequest {
  name?: string
  leader_id?: string
}

export interface GroupMemberResponse {
  group_id: string
  user_id: string
  joined_at: string
}

export interface AddMemberRequest {
  user_id: string
}
