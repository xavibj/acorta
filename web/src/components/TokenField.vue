<script setup>
import { ref } from 'vue'
import { loadToken, saveToken } from '../api.js'

// Campo Token de API y botón Guardar (spec 008, AUT-11 a AUT-13).
// El token no se valida aquí: la web no tiene reglas de negocio (spec 006).

const input = ref(null)
const token = ref(loadToken())
const saved = ref(token.value !== '')

// Guardar o Enter: guarda el token recortado; vacío borra la clave y quita el estado.
function save() {
  token.value = token.value.trim()
  saveToken(token.value)
  saved.value = token.value !== ''
}

// Lo usa App ante un 401 al crear o borrar (AUT-13).
function focus() {
  input.value?.focus()
}
defineExpose({ focus })

const inputClass =
  'block min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 text-base placeholder:text-slate-400 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 dark:border-slate-600 dark:bg-slate-900'

const secondaryButton =
  'min-h-11 shrink-0 rounded-lg border border-slate-300 px-4 font-medium transition hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:border-slate-600 dark:hover:bg-slate-800'
</script>

<template>
  <!-- Sin <form>: el formulario de la página es el de creación (AUT-11). -->
  <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-6 dark:border-slate-800 dark:bg-slate-900">
    <label for="token" class="mb-1 block text-sm font-medium">Token de API</label>
    <div class="flex gap-2">
      <input
        id="token"
        ref="input"
        v-model="token"
        type="password"
        autocomplete="off"
        :class="inputClass"
        @keydown.enter="save"
      />
      <button type="button" :class="secondaryButton" @click="save">Guardar</button>
    </div>
    <div aria-live="polite" class="empty:hidden mt-2">
      <p v-if="saved" class="text-sm text-emerald-700 dark:text-emerald-300">Token guardado</p>
    </div>
  </section>
</template>
