<script setup>
// Campo Token de API y botón Guardar (spec 008, AUT-11 a AUT-13).
// Paso rojo: todavía sin lógica.

const inputClass =
  'block min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 text-base placeholder:text-slate-400 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 disabled:opacity-60 dark:border-slate-600 dark:bg-slate-900'

const secondaryButton =
  'min-h-11 shrink-0 rounded-lg border border-slate-300 px-4 font-medium transition hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:border-slate-600 dark:hover:bg-slate-800'
</script>

<template>
  <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-6 dark:border-slate-800 dark:bg-slate-900">
    <label for="token" class="mb-1 block text-sm font-medium">Token de API</label>
    <div class="flex gap-2">
      <input id="token" type="password" autocomplete="off" :class="inputClass" />
      <button type="button" :class="secondaryButton">Guardar</button>
    </div>
  </section>
</template>
