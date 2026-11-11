<template>
  <ConversationPlaceholder
    v-if="['inbox', 'team-inbox', 'personal-inbox', 'view-inbox'].includes(route.name)"
  />
  <router-view />
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useIntervalFn, useDocumentVisibility } from '@vueuse/core'
import { useUserStore } from '@main/stores/user'
import { useConversationStore } from '../../stores/conversation'
import { CONVERSATION_LIST_TYPE } from '../../constants/conversation'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const type = computed(() => route.params.type)
const requestedStatus = computed(() => route.query.status || '')
const requestedPriority = computed(() => route.query.priority || '')
const teamID = computed(() => route.params.teamID)
const inboxID = computed(() => route.params.inboxID)
const personalListType = computed(() =>
  requestedPriority.value === 'high'
    ? CONVERSATION_LIST_TYPE.PERSONAL_HIGH
    : CONVERSATION_LIST_TYPE.PERSONAL
)
const folderID = computed(() => route.query.folder || '')
const viewID = computed(() => route.params.viewID)

const requestedListType = computed(() => {
  if (type.value === CONVERSATION_LIST_TYPE.ASSIGNED && requestedPriority.value === 'high') {
    return CONVERSATION_LIST_TYPE.ASSIGNED_HIGH
  }

  return type.value
})

const conversationStore = useConversationStore()
const userStore = useUserStore()

let lastFetchedKey = ''

const requestedFilters = computed(() => {
  const raw = route.query.filters
  if (!raw) return []

  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
})

const fetchForCurrentRoute = () => {
  if (!type.value && !teamID.value && !viewID.value && !inboxID.value) return

  const key = `${inboxID.value || ''}|${type.value || ''}|${teamID.value || ''}|${viewID.value || ''}|${requestedStatus.value}|${requestedPriority.value}|${route.query.filters || ''}|${folderID.value}`
  if (key === lastFetchedKey) return
  lastFetchedKey = key

  if (folderID.value) {
    const scope = inboxID.value
      ? 'personal'
      : teamID.value
        ? 'team'
        : type.value === 'customer'
          ? 'customer'
          : type.value === 'customer-redistributed'
            ? 'redistributed'
          : type.value === 'service-mails'
            ? 'service_mail'
          : type.value === 'assigned'
            ? 'my_inbox'
            : ''
    if (!scope) return
    conversationStore.setListStatus('', false)
    conversationStore.fetchConversationsList(
      true,
      `folder:${scope}:${scope === 'my_inbox' ? userStore.userID : inboxID.value || teamID.value || 0}:${folderID.value}`
    )
    return
  }

  if (viewID.value) {
    conversationStore.setListStatus('', false)
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], viewID.value)
    return
  }

  conversationStore.setListStatus(requestedStatus.value, false)

  if (inboxID.value) {
    conversationStore.fetchConversationsList(
      true,
      personalListType.value,
      0,
      requestedFilters.value,
      0,
      0,
      inboxID.value
    )
  } else if (type.value) {
    conversationStore.fetchConversationsList(
      true,
      requestedListType.value,
      0,
      requestedFilters.value
    )
  } else {
    const teamListType =
      requestedPriority.value === 'high'
        ? CONVERSATION_LIST_TYPE.TEAM_HIGH
        : CONVERSATION_LIST_TYPE.TEAM_ALL

    conversationStore.fetchConversationsList(
      true,
      teamListType,
      teamID.value,
      requestedFilters.value
    )
  }
}

onMounted(fetchForCurrentRoute)

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(() => conversationStore.refreshConversationList(), 120000)
watch(visibility, (v) => {
  if (v === 'visible') {
    conversationStore.refreshConversationList()
    resume()
  } else {
    pause()
  }
})

watch(
  [type, teamID, inboxID, viewID, folderID, requestedStatus, requestedPriority, requestedFilters],
  fetchForCurrentRoute
)
</script>
