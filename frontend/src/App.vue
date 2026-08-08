<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { apiFetch, apiBase, getToken, setToken, clearToken } from './lib/api'

interface User {
  id: number
  email: string
  role: 'client' | 'freelancer' | 'admin'
  display_name: string
  payout_address?: string
}

interface Order {
  ID: number
  Title: string
  AmountBTC: number
  Status: string
  Address: string
}

const health = ref<'checking' | 'ok' | 'down'>('checking')
const currentUser = ref<User | null>(null)
const isLoggedIn = computed(() => !!currentUser.value)

const authMode = ref<'login' | 'register'>('login')
const authForm = ref({ email: '', password: '', display_name: '', role: 'client' as 'client' | 'freelancer' })
const authError = ref('')

const orders = ref<Order[]>([])
const orderForm = ref({ title: '', description: '', amount_btc: 0.001 })
const payoutAddress = ref('')

async function checkHealth() {
  try {
    const res = await fetch(`${apiBase}/health`)
    health.value = res.ok ? 'ok' : 'down'
  } catch {
    health.value = 'down'
  }
}

async function loadMe() {
  if (!getToken()) return
  try {
    currentUser.value = await apiFetch('/api/me')
    payoutAddress.value = currentUser.value?.payout_address || ''
  } catch {
    clearToken()
    currentUser.value = null
  }
}

async function submitAuth() {
  authError.value = ''
  try {
    const path = authMode.value === 'login' ? '/api/auth/login' : '/api/auth/register'
    const body =
      authMode.value === 'login'
        ? { email: authForm.value.email, password: authForm.value.password }
        : authForm.value
    const res = await apiFetch(path, { method: 'POST', body: JSON.stringify(body) })
    setToken(res.token)
    currentUser.value = res.user
    payoutAddress.value = res.user.payout_address || ''
  } catch (e: any) {
    authError.value = e.message
  }
}

function logout() {
  clearToken()
  currentUser.value = null
  orders.value = []
}

async function createOrder() {
  try {
    const o = await apiFetch('/api/orders', { method: 'POST', body: JSON.stringify(orderForm.value) })
    orders.value.unshift(o)
  } catch (e: any) {
    alert('Ошибка создания заказа: ' + e.message)
  }
}

async function savePayoutAddress() {
  try {
    await apiFetch('/api/me/payout-address', {
      method: 'PUT',
      body: JSON.stringify({ address: payoutAddress.value }),
    })
    alert('Адрес для выплат сохранён')
  } catch (e: any) {
    alert('Ошибка: ' + e.message)
  }
}

onMounted(async () => {
  await checkHealth()
  await loadMe()
})
</script>

<template>
  <div class="app">
    <h1>Freelance BTC — MVP</h1>
    <p>API: <strong :class="health">{{ health }}</strong></p>

    <!-- Не залогинен: форма логина/регистрации -->
    <section v-if="!isLoggedIn" class="auth">
      <div class="tabs">
        <button :class="{ active: authMode === 'login' }" @click="authMode = 'login'">Вход</button>
        <button :class="{ active: authMode === 'register' }" @click="authMode = 'register'">Регистрация</button>
      </div>

      <input v-model="authForm.email" type="email" placeholder="Email" />
      <input v-model="authForm.password" type="password" placeholder="Пароль (мин. 8 символов)" />

      <template v-if="authMode === 'register'">
        <input v-model="authForm.display_name" placeholder="Имя / название" />
        <select v-model="authForm.role">
          <option value="client">Я заказчик</option>
          <option value="freelancer">Я фрилансер</option>
        </select>
      </template>

      <button @click="submitAuth">{{ authMode === 'login' ? 'Войти' : 'Зарегистрироваться' }}</button>
      <p v-if="authError" class="error">{{ authError }}</p>
    </section>

    <!-- Залогинен -->
    <section v-else>
      <div class="user-bar">
        <span>{{ currentUser!.display_name }} ({{ currentUser!.role }})</span>
        <button @click="logout">Выйти</button>
      </div>

      <section v-if="currentUser!.role === 'freelancer'" class="payout">
        <h2>Адрес для выплат</h2>
        <input v-model="payoutAddress" placeholder="Ваш BTC-адрес" />
        <button @click="savePayoutAddress">Сохранить</button>
      </section>

      <section class="new-order">
        <h2>Новый заказ</h2>
        <input v-model="orderForm.title" placeholder="Название заказа" />
        <textarea v-model="orderForm.description" placeholder="Описание" />
        <input v-model.number="orderForm.amount_btc" type="number" step="0.00000001" placeholder="Сумма BTC" />
        <button @click="createOrder">Создать и получить адрес для оплаты</button>
      </section>

      <section class="orders">
        <h2>Заказы (текущая сессия)</h2>
        <ul>
          <li v-for="o in orders" :key="o.ID">
            #{{ o.ID }} — {{ o.Title }} — {{ o.AmountBTC }} BTC — <em>{{ o.Status }}</em>
            <div class="address">{{ o.Address }}</div>
          </li>
        </ul>
      </section>
    </section>
  </div>
</template>

<style scoped>
.app { max-width: 640px; margin: 40px auto; font-family: system-ui, sans-serif; }
.ok { color: #16a34a; }
.down { color: #dc2626; }
.checking { color: #6b7280; }
.auth, .new-order, .payout { display: flex; flex-direction: column; gap: 8px; margin: 24px 0; }
.tabs { display: flex; gap: 8px; margin-bottom: 8px; }
.tabs button { flex: 1; background: #eee; color: #333; }
.tabs button.active { background: #f7931a; color: white; }
.user-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
input, textarea, select { padding: 8px; border: 1px solid #ccc; border-radius: 6px; }
button { padding: 10px; background: #f7931a; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: 600; }
.address { font-family: monospace; font-size: 12px; color: #555; word-break: break-all; }
.error { color: #dc2626; font-size: 14px; }
</style>
