<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-4">
        <input
          v-model="search"
          type="search"
          class="input max-w-md"
          :placeholder="t('quotaUsers.searchPlaceholder')"
          @keyup.enter="loadUsers"
        />
      </div>

      <div class="card overflow-hidden">
        <table class="w-full text-sm">
          <thead class="border-b border-gray-100 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:text-dark-400">
            <tr>
              <th class="px-5 py-3">{{ t('quotaUsers.email') }}</th>
              <th class="px-5 py-3">{{ t('quotaUsers.username') }}</th>
              <th class="px-5 py-3">{{ t('quotaUsers.balance') }}</th>
              <th class="px-5 py-3 text-right">{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="px-5 py-10 text-center text-gray-500">{{ t('common.loading') }}</td>
            </tr>
            <tr v-else-if="users.length === 0">
              <td colspan="4" class="px-5 py-10 text-center text-gray-500">{{ t('quotaUsers.empty') }}</td>
            </tr>
            <tr
              v-for="user in users"
              :key="user.id"
              class="border-b border-gray-50 last:border-0 dark:border-dark-800"
            >
              <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">{{ user.email }}</td>
              <td class="px-5 py-3 text-gray-600 dark:text-dark-300">{{ user.username || '-' }}</td>
              <td class="px-5 py-3 font-mono">${{ formatBalance(user.balance) }}</td>
              <td class="px-5 py-3 text-right">
                <button class="btn btn-secondary mr-2" @click="openModal(user, 'add')">{{ t('quotaUsers.add') }}</button>
                <button class="btn btn-danger" @click="openModal(user, 'subtract')">{{ t('quotaUsers.subtract') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="showModal && activeUser" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="card w-full max-w-md p-6">
        <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
          {{ operation === 'add' ? t('quotaUsers.add') : t('quotaUsers.subtract') }}
        </h3>
        <p class="mb-4 text-sm text-gray-500">
          {{ activeUser.email }} · {{ t('quotaUsers.currentBalance') }} ${{ formatBalance(activeUser.balance) }}
        </p>
        <label class="input-label">{{ t('quotaUsers.amount') }}</label>
        <input v-model.number="amount" type="number" min="0" step="any" class="input mb-4" />
        <label class="input-label">{{ t('quotaUsers.notes') }}</label>
        <textarea v-model="notes" rows="3" class="input mb-4"></textarea>
        <p v-if="amount > 0" class="mb-4 text-sm">
          {{ t('quotaUsers.newBalance') }}:
          <span class="font-semibold">${{ formatBalance(previewBalance) }}</span>
        </p>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="showModal = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="submitting || !amount" @click="submit">
            {{ submitting ? t('common.saving') : t('common.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores/app'
import { listUsers, updateBalance, type QuotaUser } from '@/api/quota'

const { t } = useI18n()
const appStore = useAppStore()
const users = ref<QuotaUser[]>([])
const loading = ref(false)
const search = ref('')
const showModal = ref(false)
const activeUser = ref<QuotaUser | null>(null)
const operation = ref<'add' | 'subtract'>('add')
const amount = ref(0)
const notes = ref('')
const submitting = ref(false)

const previewBalance = computed(() => {
  if (!activeUser.value) return 0
  const next = operation.value === 'add'
    ? activeUser.value.balance + amount.value
    : activeUser.value.balance - amount.value
  return Math.abs(next) < 1e-10 ? 0 : next
})

function formatBalance(value: number) {
  return value.toFixed(2)
}

async function loadUsers() {
  loading.value = true
  try {
    const data = await listUsers(1, 100, search.value)
    users.value = data.items || []
  } catch (error) {
    console.error(error)
    appStore.showError(t('common.error'))
  } finally {
    loading.value = false
  }
}

function openModal(user: QuotaUser, op: 'add' | 'subtract') {
  activeUser.value = user
  operation.value = op
  amount.value = 0
  notes.value = ''
  showModal.value = true
}

async function submit() {
  if (!activeUser.value || !amount.value || amount.value <= 0) {
    appStore.showError(t('quotaUsers.amountRequired'))
    return
  }
  if (operation.value === 'subtract' && amount.value > activeUser.value.balance) {
    appStore.showError(t('quotaUsers.insufficientBalance'))
    return
  }
  submitting.value = true
  try {
    await updateBalance(activeUser.value.id, amount.value, operation.value, notes.value)
    appStore.showSuccess(t('common.success'))
    showModal.value = false
    await loadUsers()
  } catch (error) {
    console.error(error)
    appStore.showError(t('common.error'))
  } finally {
    submitting.value = false
  }
}

onMounted(loadUsers)
</script>
