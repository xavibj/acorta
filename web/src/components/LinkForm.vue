<script setup>
import { computed, ref } from 'vue'
import { createLink } from '../api.js'
import CopyButton from './CopyButton.vue'

// unauthorized: la API ha respondido 401 (spec 008, AUT-13); App enfoca el campo del token.
const emit = defineEmits(['created', 'unauthorized'])

// Plazos del desplegable, en segundos. «Nunca» no envía expires_at.
const EXPIRIES = [
  { label: 'Nunca', seconds: 0 },
  { label: '1 hora', seconds: 3600 },
  { label: '1 día', seconds: 86400 },
  { label: '7 días', seconds: 7 * 86400 },
  { label: '30 días', seconds: 30 * 86400 },
]

const url = ref('')
const alias = ref('')
const expiry = ref(0)
const busy = ref(false)
const errors = ref([])
const created = ref(null)

const canSubmit = computed(() => url.value !== '' && !busy.value)

// Al editar un campo se descartan los errores y el resultado anteriores (WEB-06).
function clearFeedback() {
  errors.value = []
  created.value = null
}

// RFC 3339 en UTC y a precisión de segundos (spec 001).
function expiresAt(seconds) {
  const date = new Date(Date.now() + seconds * 1000)
  return date.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

async function submit() {
  if (!canSubmit.value) return
  const payload = { url: url.value }
  if (alias.value !== '') payload.alias = alias.value
  if (expiry.value > 0) payload.expires_at = expiresAt(expiry.value)

  busy.value = true
  clearFeedback()
  const result = await createLink(payload)
  busy.value = false

  if (!result.ok) {
    errors.value = result.errors
    if (result.status === 401) emit('unauthorized')
    return
  }
  created.value = result.data
  url.value = ''
  alias.value = ''
  expiry.value = 0
  emit('created', result.data)
}

const inputClass =
  'block min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 text-base placeholder:text-slate-400 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 disabled:opacity-60 dark:border-slate-600 dark:bg-slate-900'
</script>

<template>
  <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-6 dark:border-slate-800 dark:bg-slate-900">
    <form class="space-y-4" @submit.prevent="submit">
      <div>
        <label for="url" class="mb-1 block text-sm font-medium">URL</label>
        <input
          id="url"
          v-model="url"
          type="text"
          inputmode="url"
          autocomplete="off"
          placeholder="https://example.com/una/ruta/larga"
          :class="inputClass"
          :disabled="busy"
          @input="clearFeedback"
        />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label for="alias" class="mb-1 block text-sm font-medium">Alias (opcional)</label>
          <input
            id="alias"
            v-model="alias"
            type="text"
            autocomplete="off"
            placeholder="promo"
            :class="inputClass"
            :disabled="busy"
            @input="clearFeedback"
          />
        </div>
        <div>
          <label for="expiry" class="mb-1 block text-sm font-medium">Caducidad</label>
          <select id="expiry" v-model="expiry" :class="inputClass" :disabled="busy" @change="clearFeedback">
            <option v-for="e in EXPIRIES" :key="e.label" :value="e.seconds">{{ e.label }}</option>
          </select>
        </div>
      </div>
      <button
        type="submit"
        class="min-h-11 w-full rounded-lg bg-indigo-600 px-5 font-semibold text-white transition hover:bg-indigo-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
        :disabled="!canSubmit"
      >
        Acortar
      </button>
    </form>

    <div aria-live="polite" class="empty:hidden mt-4">
      <ul
        v-if="errors.length"
        class="space-y-1 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-800 dark:bg-red-950 dark:text-red-200"
      >
        <li v-for="e in errors" :key="e">{{ e }}</li>
      </ul>
      <div
        v-if="created"
        class="flex items-center gap-3 rounded-lg bg-indigo-50 p-3 dark:bg-indigo-950"
      >
        <p class="min-w-0 flex-1 break-all text-lg font-semibold text-indigo-700 dark:text-indigo-300">
          {{ created.short_url }}
        </p>
        <CopyButton :text="created.short_url" />
      </div>
    </div>
  </section>
</template>
