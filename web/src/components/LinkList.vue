<script setup>
import CopyButton from './CopyButton.vue'

defineProps({
  links: { type: Array, required: true },
  loading: Boolean,
  loadFailed: Boolean,
  errors: { type: Array, default: () => [] },
})
defineEmits(['refresh', 'delete'])

function visitsLabel(n) {
  return n === 1 ? '1 visita' : `${n} visitas`
}

// dd/mm/aaaa hh:mm en la hora local del navegador.
function formatDate(iso) {
  const d = new Date(iso)
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getDate())}/${p(d.getMonth() + 1)}/${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const secondaryButton =
  'min-h-11 shrink-0 rounded-lg border border-slate-300 px-4 font-medium transition hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:border-slate-600 dark:hover:bg-slate-800'
</script>

<template>
  <section>
    <div class="mb-3 flex items-center justify-between gap-3">
      <h2 class="text-xl font-semibold">Tus enlaces</h2>
      <button type="button" :class="secondaryButton" @click="$emit('refresh')">Actualizar</button>
    </div>

    <div aria-live="polite" class="empty:hidden">
      <ul
        v-if="errors.length"
        class="mb-3 space-y-1 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-800 dark:bg-red-950 dark:text-red-200"
      >
        <li v-for="e in errors" :key="e">{{ e }}</li>
      </ul>
      <div
        v-if="loadFailed"
        class="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-red-50 px-4 py-3 text-red-800 dark:bg-red-950 dark:text-red-200"
      >
        <p>No se ha podido cargar la lista de enlaces.</p>
        <button type="button" :class="secondaryButton" @click="$emit('refresh')">Reintentar</button>
      </div>
      <p v-else-if="loading && !links.length" class="text-slate-500 dark:text-slate-400">Cargando…</p>
      <p v-else-if="!links.length" class="text-slate-600 dark:text-slate-400">
        Aún no hay enlaces. Crea el primero arriba.
      </p>
    </div>

    <ul v-if="!loadFailed && links.length" class="space-y-3">
      <li
        v-for="link in links"
        :key="link.code"
        class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900"
      >
        <a
          :href="link.short_url"
          target="_blank"
          rel="noopener"
          class="break-all text-lg font-semibold text-indigo-700 hover:underline dark:text-indigo-300"
        >
          {{ link.short_url }}
        </a>
        <p class="mt-1 truncate text-sm text-slate-600 dark:text-slate-400" :title="link.url">{{ link.url }}</p>
        <!-- espacio real entre bloques: así el texto copiado o leído no pega la URL con las visitas -->
        {{ ' ' }}
        <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2">
          <p class="min-w-0 flex-1 text-sm text-slate-600 dark:text-slate-400">
            <span>{{ visitsLabel(link.visits) }}</span>{{ ' ' }}
            <span v-if="link.expired" class="ml-3 rounded bg-amber-100 px-2 py-0.5 font-medium text-amber-800 dark:bg-amber-900 dark:text-amber-100">Caducado</span>
            <span v-else-if="link.expires_at" class="ml-3">Caduca el {{ formatDate(link.expires_at) }}</span>
          </p>
          <div class="flex gap-2">
            <CopyButton :text="link.short_url" />
            <button type="button" :class="secondaryButton" @click="$emit('delete', link.code)">Borrar</button>
          </div>
        </div>
      </li>
    </ul>
  </section>
</template>
