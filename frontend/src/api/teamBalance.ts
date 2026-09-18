import { apiClient } from './client'

export interface TeamBalance {
  total_budget: number
  consumed: number
  remaining: number
  revision: number
}

export async function getTeamBalance(admin = false, signal?: AbortSignal): Promise<TeamBalance> {
  const { data } = await apiClient.get<TeamBalance>(admin ? '/admin/team-balance' : '/user/team-balance', { signal })
  return data
}

export async function adjustTeamBalance(operation: 'set' | 'add', amount: number, revision: number): Promise<TeamBalance> {
  const { data } = await apiClient.post<TeamBalance>('/admin/team-balance', { operation, amount, revision })
  return data
}
