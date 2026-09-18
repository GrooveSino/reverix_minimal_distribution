import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import TeamBalanceCard from '../TeamBalanceCard.vue'
import { adjustTeamBalance, getTeamBalance } from '@/api/teamBalance'

vi.mock('@/api/teamBalance', () => ({ getTeamBalance: vi.fn(), adjustTeamBalance: vi.fn() }))
let wrapper: VueWrapper | undefined
const state = { total_budget: 10000, consumed: 1000, remaining: 9000, revision: 1 }

beforeEach(() => {
  vi.useFakeTimers()
  vi.resetAllMocks()
  vi.mocked(getTeamBalance).mockResolvedValue({ ...state })
})
afterEach(() => { wrapper?.unmount(); vi.useRealTimers() })

describe('team balance', () => {
  it('shows shared balance to members without administrative controls', async () => {
    wrapper = mount(TeamBalanceCard)
    await flushPromises()
    expect(wrapper.text()).toContain('$9,000.00')
    expect(wrapper.find('form').exists()).toBe(false)
    expect(getTeamBalance).toHaveBeenCalledWith(false, expect.any(AbortSignal))
    vi.mocked(getTeamBalance).mockResolvedValue({ ...state, consumed: 1100, remaining: 8900 })
    await vi.advanceTimersByTimeAsync(10000)
    expect(wrapper.text()).toContain('$8,900.00')
  })

  it('shows exhaustion including negative settlement amounts and then recovery', async () => {
    vi.mocked(getTeamBalance).mockResolvedValue({ ...state, consumed: 10100, remaining: -100 })
    wrapper = mount(TeamBalanceCard)
    await flushPromises()
    expect(wrapper.text()).toContain('所有新调用已暂停')
    expect(wrapper.text()).toContain('-$100.00')
    vi.mocked(getTeamBalance).mockResolvedValue({ ...state })
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('所有新调用已暂停')
  })

  it('adds funds with the loaded version and does not change personal balances', async () => {
    vi.mocked(adjustTeamBalance).mockResolvedValue({ ...state, total_budget: 15000, remaining: 14000, revision: 2 })
    wrapper = mount(TeamBalanceCard, { props: { admin: true } })
    await flushPromises()
    await wrapper.get('input').setValue('5000')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(adjustTeamBalance).toHaveBeenCalledTimes(1)
    expect(adjustTeamBalance).toHaveBeenCalledWith('add', 5000, 1)
    expect(wrapper.text()).toContain('$14,000.00')
  })

  it('keeps the original revision for an edit even when polling sees another admin update', async () => {
    wrapper = mount(TeamBalanceCard, { props: { admin: true } })
    await flushPromises()
    await wrapper.get('select').setValue('set')
    await wrapper.get('input').setValue('12000')
    vi.mocked(getTeamBalance).mockResolvedValue({ ...state, total_budget: 15000, remaining: 14000, revision: 2 })
    await vi.advanceTimersByTimeAsync(10000)
    vi.mocked(adjustTeamBalance).mockRejectedValue(new Error('公池额度已被修改，请刷新后重试。'))
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(adjustTeamBalance).toHaveBeenCalledTimes(1)
    expect(adjustTeamBalance).toHaveBeenCalledWith('set', 12000, 1)
    expect(wrapper.get('[role="alert"]').text()).toContain('公池额度已被修改')
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
  })

  it('marks old values stale after a read failure and prevents edits', async () => {
    wrapper = mount(TeamBalanceCard, { props: { admin: true } })
    await flushPromises()
    vi.mocked(getTeamBalance).mockRejectedValue(new Error('network'))
    await vi.advanceTimersByTimeAsync(10000)
    await wrapper.get('input').setValue('100')
    expect(wrapper.text()).toContain('上次读取的剩余余额')
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit')
    expect(adjustTeamBalance).not.toHaveBeenCalled()
  })
})
