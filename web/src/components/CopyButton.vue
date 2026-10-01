<script setup>
import { onBeforeUnmount, ref } from 'vue'

const props = defineProps({ text: { type: String, required: true } })

const copied = ref(false)
let timer = null

async function copy() {
  try {
    await navigator.clipboard.writeText(props.text)
  } catch {
    return // sin portapapeles disponible: no fingimos que se ha copiado
  }
  copied.value = true
  clearTimeout(timer)
  timer = setTimeout(() => (copied.value = false), 2000)
}

onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <button
    type="button"
    class="min-h-11 shrink-0 rounded-lg border border-slate-300 px-4 font-medium transition hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:border-slate-600 dark:hover:bg-slate-800"
    @click="copy"
  >
    {{ copied ? 'Copiado' : 'Copiar' }}
  </button>
</template>
