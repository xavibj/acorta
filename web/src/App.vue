<script setup>
import { onMounted, ref } from 'vue'
import { deleteLink, listLinks } from './api.js'
import LinkForm from './components/LinkForm.vue'
import LinkList from './components/LinkList.vue'
import TokenField from './components/TokenField.vue'

const links = ref([])
const loading = ref(true)
const loadFailed = ref(false)
const deleteErrors = ref([])

async function load() {
  loading.value = true
  loadFailed.value = false
  const result = await listLinks()
  if (result.ok) links.value = result.data
  else loadFailed.value = true
  loading.value = false
}

function onCreated(link) {
  links.value.unshift(link)
}

async function onDelete(code) {
  deleteErrors.value = []
  if (!window.confirm(`¿Borrar /${code}?`)) return
  const result = await deleteLink(code)
  if (result.ok) links.value = links.value.filter((l) => l.code !== code)
  else deleteErrors.value = result.errors
}

onMounted(load)
</script>

<template>
  <div class="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
    <main class="mx-auto w-full max-w-2xl space-y-6 px-4 py-8 md:py-12">
      <header>
        <h1 class="text-3xl font-bold tracking-tight">acorta</h1>
        <p class="mt-1 text-slate-600 dark:text-slate-400">Acorta una URL larga y compártela.</p>
      </header>
      <TokenField />
      <LinkForm @created="onCreated" />
      <LinkList
        :links="links"
        :loading="loading"
        :load-failed="loadFailed"
        :errors="deleteErrors"
        @refresh="load"
        @delete="onDelete"
      />
    </main>
  </div>
</template>
