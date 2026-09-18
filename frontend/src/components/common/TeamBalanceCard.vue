<template>
  <section class="card overflow-hidden" aria-labelledby="team-balance-heading">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 p-5 dark:border-dark-700">
      <div>
        <h2 id="team-balance-heading" class="text-base font-semibold text-gray-900 dark:text-white">团队公池余额</h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">全体成员共享 · 每笔消费同时扣减个人余额和公池余额</p>
      </div>
      <button type="button" class="btn btn-secondary text-sm" :disabled="loading || saving" @click="load">刷新余额</button>
    </div>
    <div class="p-5">
      <p v-if="error" role="alert" class="mb-3 text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <p v-if="!balance && loading" class="text-sm text-gray-500">正在读取公池余额…</p>
      <template v-if="balance">
        <div class="grid gap-5 sm:grid-cols-3" :class="{ 'opacity-50': stale }">
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ stale ? '上次读取的剩余余额' : '当前剩余余额' }}</p>
            <p class="mt-1 text-3xl font-bold tabular-nums" :class="balance.remaining > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'">{{ usd(balance.remaining) }}</p>
          </div>
          <div><p class="text-xs text-gray-500 dark:text-gray-400">累计总额度</p><p class="mt-2 text-xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ usd(balance.total_budget) }}</p></div>
          <div><p class="text-xs text-gray-500 dark:text-gray-400">团队累计消费（含历史）</p><p class="mt-2 text-xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ usd(balance.consumed) }}</p></div>
        </div>
        <p v-if="balance.remaining <= 0 && !stale" role="status" class="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
          公池余额已用完，所有新调用已暂停。请联系管理员增加额度；已开始的调用会继续完成并结算。
        </p>
        <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">余额每 10 秒自动刷新。个人余额不足时，仅该成员暂停调用。</p>
      </template>

      <form v-if="admin && balance" class="mt-5 space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700" @submit.prevent="save">
        <div class="flex flex-wrap items-end gap-3">
          <label class="space-y-1 text-sm text-gray-700 dark:text-gray-300">
            <span class="block">额度操作</span>
            <select v-model="operation" class="input" :disabled="saving" @change="amount = ''; success = ''">
              <option value="add">追加额度</option>
              <option value="set">设置累计总额度</option>
            </select>
          </label>
          <label class="space-y-1 text-sm text-gray-700 dark:text-gray-300">
            <span class="block">{{ operation === 'add' ? '追加金额（USD）' : '累计总额度（USD）' }}</span>
            <input v-model="amount" class="input" type="number" step="0.00000001" :min="operation === 'add' ? 0.00000001 : 0" max="10000000000" required :disabled="saving" placeholder="例如 10000" />
          </label>
          <button type="submit" class="btn btn-primary" :disabled="!validAmount || saving || loading || stale">{{ saving ? '保存中…' : operation === 'add' ? '追加额度' : '保存总额度' }}</button>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">设置的是累计总额度，不是剩余余额；历史消费始终保留。修改公池不会给个人账户充值。</p>
        <p v-if="validAmount" class="text-sm text-gray-700 dark:text-gray-300">按当前消费计算，保存后剩余约 {{ usd(projectedRemaining) }}<span v-if="projectedRemaining <= 0" class="text-red-600">，新调用仍将暂停</span>。</p>
        <p v-if="success" role="status" class="text-sm text-emerald-600 dark:text-emerald-400">{{ success }}</p>
      </form>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { adjustTeamBalance, getTeamBalance, type TeamBalance } from '@/api/teamBalance'

const props = withDefaults(defineProps<{ admin?: boolean }>(), { admin: false })
const balance = ref<TeamBalance | null>(null)
const loading = ref(false)
const saving = ref(false)
const stale = ref(false)
const error = ref('')
const success = ref('')
const operation = ref<'set' | 'add'>('add')
const amount = ref<string | number>('')
let timer: ReturnType<typeof setInterval> | undefined
let disposed = false
let request: AbortController | undefined
// Freeze the admin revision while a draft exists, so polling cannot silently
// accept another administrator's update and overwrite it with an older draft.
const draftRevision = ref<number | null>(null)
const numericAmount = computed(() => Number(amount.value))
const validAmount = computed(() => amount.value !== '' && Number.isFinite(numericAmount.value) && numericAmount.value <= 1e10 && (operation.value === 'add' ? numericAmount.value > 0 : numericAmount.value >= 0))
const projectedRemaining = computed(() => (operation.value === 'add' ? (balance.value?.total_budget ?? 0) : 0) + numericAmount.value - (balance.value?.consumed ?? 0))
const usd = (value: number) => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 8 }).format(value)

async function load() {
  if (loading.value || saving.value || disposed) return
  loading.value = true
  request = new AbortController()
  try {
    const result = await getTeamBalance(props.admin, request.signal)
    if (disposed) return
    balance.value = result
    if (amount.value === '' || draftRevision.value === null) draftRevision.value = result.revision
    stale.value = false
    error.value = ''
  } catch {
    if (!disposed) { stale.value = true; error.value = '公池余额读取失败，请刷新重试。' }
  } finally { loading.value = false }
}

async function save() {
  if (!balance.value || !validAmount.value || saving.value || loading.value || stale.value) return
  saving.value = true
  success.value = ''
  error.value = ''
  try {
    balance.value = await adjustTeamBalance(operation.value, numericAmount.value, draftRevision.value ?? balance.value.revision)
    draftRevision.value = balance.value.revision
    amount.value = ''
    success.value = '公池额度已更新，所有成员立即生效。'
  } catch (e: unknown) {
    // Do not retry a financial mutation automatically. Refresh/reconcile first.
    error.value = (e as { message?: string })?.message || '保存未确认，请刷新核对额度后再操作。'
    stale.value = true
    amount.value = ''
    draftRevision.value = null
  } finally { saving.value = false }
}

function refreshVisible() { if (!document.hidden) void load() }
onMounted(() => {
  void load()
  timer = setInterval(refreshVisible, 10000)
  document.addEventListener('visibilitychange', refreshVisible)
})
onUnmounted(() => {
  disposed = true
  request?.abort()
  if (timer) clearInterval(timer)
  document.removeEventListener('visibilitychange', refreshVisible)
})
</script>
