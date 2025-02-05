<template>
  <div class="overflow-y-scroll h-full">
    <SearchHeader v-model="searchQuery" @search="handleSearch" />

    <div v-if="loading" class="flex justify-center items-center h-64">
      <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
    </div>

    <div v-else-if="error" class="mt-8 text-center">
      <p class="text-lg text-destructive">{{ error }}</p>
      <button
        @click="handleSearch"
        class="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
      >
        Try Again
      </button>
    </div>

    <div v-else>
      <p
        v-if="searchPerformed && totalResults === 0"
        class="mt-8 text-center text-muted-foreground"
      >
        No results found for "{{ searchQuery }}". Try a different search term.
      </p>
      <SearchResults v-else-if="searchPerformed" :results="results" />

      <p
        v-else-if="searchQuery.length > 0 && searchQuery.length < MIN_SEARCH_LENGTH"
        class="mt-8 text-center text-muted-foreground"
      >
        Please enter at least {{ MIN_SEARCH_LENGTH }} characters to search.
      </p>

      <!-- New component for when search is not performed -->
      <div v-else class="mt-16 text-center">
        <h2 class="text-2xl font-semibold text-primary mb-4">Search conversations</h2>
        <p class="text-lg text-muted-foreground">
          Search by reference number, messages, or any keywords related to your conversations.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import SearchHeader from '@/features/search/SearchHeader.vue'
import SearchResults from '@/features/search/SearchResults.vue'
import api from '@/api'

const MIN_SEARCH_LENGTH = 2
const DEBOUNCE_DELAY = 300

const searchQuery = ref('')
const results = ref({ conversations: [], messages: [] })
const loading = ref(false)
const error = ref(null)
const searchPerformed = ref(false)
let debounceTimer = null

const totalResults = computed(() => {
  return results.value.conversations.length + results.value.messages.length
})

const handleSearch = async () => {
  if (searchQuery.value.length < MIN_SEARCH_LENGTH) {
    results.value = { conversations: [], messages: [] }
    searchPerformed.value = false
    return
  }

  loading.value = true
  error.value = null
  searchPerformed.value = true

  try {
    const [convResults, messagesResults] = await Promise.all([
      api.searchConversations({ query: searchQuery.value }),
      api.searchMessages({ query: searchQuery.value })
    ])

    results.value = {
      conversations: convResults.data.data,
      messages: messagesResults.data.data
    }
  } catch (err) {
    console.error(err)
    error.value = 'An error occurred while searching. Please try again.'
  } finally {
    loading.value = false
  }
}

const debouncedSearch = () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(handleSearch, DEBOUNCE_DELAY)
}

watch(searchQuery, (newValue) => {
  if (newValue.length >= MIN_SEARCH_LENGTH) {
    debouncedSearch()
  } else {
    clearTimeout(debounceTimer)
    results.value = { conversations: [], messages: [] }
    searchPerformed.value = false
  }
})

onBeforeUnmount(() => {
  clearTimeout(debounceTimer)
})
</script>
