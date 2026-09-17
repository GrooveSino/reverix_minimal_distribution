import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export interface QuotaUser {
  id: number
  email: string
  username: string
  balance: number
  status: string
}

export async function listUsers(page = 1, pageSize = 20, search = ''): Promise<PaginatedResponse<QuotaUser>> {
  const { data } = await apiClient.get<PaginatedResponse<QuotaUser>>('/quota/users', {
    params: { page, page_size: pageSize, search }
  })
  return data
}

export async function updateBalance(
  id: number,
  balance: number,
  operation: 'set' | 'add' | 'subtract',
  notes = ''
): Promise<QuotaUser> {
  const { data } = await apiClient.post<QuotaUser>(`/quota/users/${id}/balance`, {
    balance,
    operation,
    notes
  })
  return data
}
