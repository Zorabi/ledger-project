export type Role = 'super_admin' | 'family_admin' | 'member'

export type ViewKey =
  | 'dashboard'
  | 'todos'
  | 'families'
  | 'users'
  | 'contacts'
  | 'records'
  | 'dictionaries'
  | 'stats'
  | 'receivables'
  | 'payables'
  | 'nonreciprocal'
  | 'backup'
  | 'imports'
  | 'audit'

export interface SessionUser {
  id: number
  family_id?: number
  username: string
  display_name: string
  email: string
  role: Role
  must_change_password: boolean
  family?: { id: number; name: string }
}

export interface PageResult<T> {
  total: number
  list: T[]
  page: number
  page_size: number
}
