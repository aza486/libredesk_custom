<script setup>
import api from '@main/api'
import {
  adminNavItems,
  reportsNavItems,
  accountNavItems,
  contactNavItems
} from '../../constants/navigation'
import { useRoute, useRouter } from 'vue-router'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarProvider
} from '@shared-ui/components/ui/sidebar'
import { useAppSettingsStore } from '@main/stores/appSettings'
import {
  ChevronRight,
  User,
  Search,
  Plus,
  CircleDashed,
  List,
  Settings,
  Clock,
  Timer,
  Inbox as InboxIcon,
  CircleDot,
  Tag,
  SlidersHorizontal,
  Eye,
  Zap,
  Workflow,
  UserRound,
  UsersRound,
  Shield,
  ScrollText,
  Mail,
  MailCheck,
  FileText,
  KeyRound,
  Webhook,
  Link,
  BarChart3,
  CircleUser,
  Contact,
  FilePlus,
  Sparkles,
  NotebookText,
  Wrench,
  Bot,
  Lightbulb,
  BookOpen
} from 'lucide-vue-next'

const navIconMap = {
  Settings,
  Clock,
  Timer,
  Inbox: InboxIcon,
  CircleDot,
  Tag,
  SlidersHorizontal,
  Eye,
  Zap,
  Workflow,
  UserRound,
  UsersRound,
  Shield,
  ScrollText,
  Mail,
  MailCheck,
  FileText,
  KeyRound,
  Webhook,
  Link,
  BarChart3,
  CircleUser,
  Contact,
  Sparkles,
  NotebookText,
  Wrench,
  Bot,
  Lightbulb,
  BookOpen
}

import MobileDrawerNav from './MobileDrawerNav.vue'
import MobileDrawerFooter from './MobileDrawerFooter.vue'
import { filterNavItems } from '@main/utils/nav-permissions'
import { useStorage } from '@vueuse/core'
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@main/stores/user'
import { useConversationStore } from '@main/stores/conversation'
import UnreadCountBadge from '@main/components/UnreadCountBadge.vue'
import { useIsMobile } from '@shared-ui/composables'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'

const props = defineProps({
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] },
  sharedViews: { type: Array, default: () => [] }
})

const userStore = useUserStore()
const conversationStore = useConversationStore()
const emitter = useEmitter()

const settingsStore = useAppSettingsStore()
const route = useRoute()
const router = useRouter()
const isMobile = useIsMobile()
const { t } = useI18n()

const emit = defineEmits(['createView', 'editView', 'deleteView', 'createConversation'])

const isActiveParent = (parentHref) => {
  return route.path.startsWith(parentHref)
}

const isInboxRoute = (path) => {
  return path.startsWith('/inboxes')
}

const keepConversationOpen = () =>
  !isMobile.value &&
  conversationStore.isConversationOpen &&
  Boolean(conversationStore.conversation.data?.uuid)

const navigateToInbox = (type, query = {}) => {
  if (keepConversationOpen()) {
    router.push({
      name: 'inbox-conversation',
      params: {
        type,
        uuid: conversationStore.conversation.data.uuid
      },
      query
    })
  } else {
    router.push({
      name: 'inbox',
      params: { type },
      query
    })
  }
}

const navigateToTeamInbox = (teamID, query = {}) => {
  if (keepConversationOpen()) {
    router.push({
      name: 'team-inbox-conversation',
      params: {
        teamID,
        uuid: conversationStore.conversation.data.uuid
      },
      query
    })
  } else {
    router.push({
      name: 'team-inbox',
      params: { teamID },
      query
    })
  }
}

const navigateToPersonalInbox = (inboxID, query) => {
  router.push({
    name: keepConversationOpen() ? 'personal-inbox-conversation' : 'personal-inbox',
    params: {
      inboxID,
      ...(keepConversationOpen() && { uuid: conversationStore.conversation.data.uuid })
    },
    query
  })
}

const filteredAdminNavItems = computed(() => filterNavItems(adminNavItems, userStore.can))
const filteredReportsNavItems = computed(() => filterNavItems(reportsNavItems, userStore.can))
const filteredContactsNavItems = computed(() => filterNavItems(contactNavItems, userStore.can))

// For auto opening admin collapsibles when a child route is active
const openAdminCollapsible = ref(null)

const toggleAdminCollapsible = (titleKey) => {
  openAdminCollapsible.value = openAdminCollapsible.value === titleKey ? null : titleKey
}

// Watch for route changes and update the active collapsible
watch(
  [() => route.path, filteredAdminNavItems],
  () => {
    const activeItem = filteredAdminNavItems.value.find((item) => {
      if (!item.children) return isActiveParent(item.href)
      return item.children.some((child) => isActiveParent(child.href))
    })

    if (activeItem) {
      openAdminCollapsible.value = activeItem.titleKey
    }
  },
  { immediate: true }
)

// Sidebar open state in local storage
const sidebarOpen = useStorage('mainSidebarOpen', true)
const teamInboxOpen = useStorage('teamInboxOpen', true)
const personalInboxes = ref([])
const personalInboxOpen = useStorage('personalInboxOpen', true)
const personalInboxOpenStates = useStorage('personalInboxOpenStates', {})
const myTicketsOpen = useStorage('myTicketsOpen', true)
const customerTicketsOpen = useStorage('customerTicketsOpen', true)

const isCustomerSupport = computed(() => userStore.roles.includes('Kundensupport'))

const isAdmin = computed(() => userStore.roles.includes('Admin'))

// Individual team collapsible state
const teamInboxOpenStates = useStorage('teamInboxOpenStates', {})

const isTeamInboxOpen = (teamID) => teamInboxOpenStates.value[teamID] !== false

const setTeamInboxOpen = (teamID, open) => {
  teamInboxOpenStates.value = {
    ...teamInboxOpenStates.value,
    [teamID]: open
  }
}

const isTeamRouteActive = (teamID, status, priority = '') => {
  return (
    String(route.params.teamID) === String(teamID) &&
    route.query.status === status &&
    (route.query.priority || '') === priority
  )
}

// Spam uses existing system tag ID 12 (🗑Spam)
const spamFilter = JSON.stringify([
  {
    model: 'conversations',
    field: 'tags',
    operator: 'contains',
    value: JSON.stringify([12])
  }
])

// Einheitliche Reihenfolge in allen Bereichen:
// Hohe Priorität > Offen > Beantwortet > Schlummernd > Geschlossen > Spam
// Reuse the customer inbox's tag filter and the team inbox's status queries.
const personalInboxItems = computed(() => [
  {
    key: 'high',
    label: t('inbox.personalHighPriority'),
    query: { status: 'Open', priority: 'high' },
    count: true
  },
  { key: 'open', label: t('globals.terms.open'), query: { status: 'Open' }, count: true },
  { key: 'resolved', label: t('inbox.personalAnswered'), query: { status: 'Resolved' } },
  { key: 'snoozed', label: t('globals.terms.snoozed'), query: { status: 'Snoozed' } },
  { key: 'closed', label: t('globals.terms.closed'), query: { status: 'Closed' } },
  { key: 'spam', label: t('inbox.personalSpam'), query: { filters: spamFilter } }
])

const isPersonalRouteActive = (inboxID, query) =>
  String(route.params.inboxID) === String(inboxID) &&
  ['status', 'priority', 'filters'].every((key) => (route.query[key] || '') === (query[key] || ''))

const sidebarCounts = ref({})
let sidebarCountInterval = null
let loadingSidebarCounts = false
const handleInboxRefresh = (event) => {
  if (event?.model && ['inbox', 'inbox-list', 'personal-inbox'].includes(event.model)) {
    loadSidebarCounts()
  }
}

const loadSidebarCounts = async () => {
  if (loadingSidebarCounts) return
  loadingSidebarCounts = true
  try {
    const ownInboxesResponse = await api.getOwnPersonalInboxes()
    personalInboxes.value = ownInboxesResponse.data.data
    await Promise.all(
      personalInboxes.value.flatMap((inbox) =>
        personalInboxItems.value
          .filter((item) => item.count)
          .map(async (item) => {
            const filters =
              item.query.filters ||
              JSON.stringify([
                {
                  model: 'conversation_statuses',
                  field: 'name',
                  operator: 'equals',
                  value: item.query.status
                }
              ])
            const response = await api.getPersonalConversations(inbox.id, {
              page: 1,
              page_size: 1,
              filters,
              ...(item.query.priority && { priority: item.query.priority })
            })
            sidebarCounts.value[`personal_${inbox.id}_${item.key}`] = response.data.data.total || 0
          })
      )
    )

    const openFilter = JSON.stringify([
      {
        model: 'conversation_statuses',
        field: 'name',
        operator: 'equals',
        value: 'Open'
      }
    ])

    // Meine Tickets
    const [assignedResponse, assignedHighResponse] = await Promise.all([
      api.getAssignedConversations({
        page: 1,
        page_size: 1,
        filters: openFilter
      }),
      api.getAssignedConversations({
        page: 1,
        page_size: 1,
        priority: 'high',
        filters: openFilter
      })
    ])

    sidebarCounts.value.assigned = assignedResponse.data.data.total || 0

    sidebarCounts.value.assigned_high = assignedHighResponse.data.data.total || 0

    // Kundentickets
    if (isCustomerSupport.value || isAdmin.value) {
      const [customerResponse, highPriorityResponse] = await Promise.all([
        api.getCustomerConversations({
          page: 1,
          page_size: 1,
          filters: openFilter
        }),
        api.getCustomerConversations({
          page: 1,
          page_size: 1,
          priority: 'high',
          filters: openFilter
        })
      ])

      sidebarCounts.value.customer = customerResponse.data.data.total || 0

      sidebarCounts.value.customer_high = highPriorityResponse.data.data.total || 0
    }

    // Admin-only unassigned
    if (isAdmin.value) {
      const unassignedResponse = await api.getUnassignedConversations({
        page: 1,
        page_size: 100
      })

      sidebarCounts.value.unassigned = unassignedResponse.data.data.results.filter(
        (conversation) => conversation.status === 'Open'
      ).length
    }

    // Teams
    for (const team of props.userTeams || []) {
      const [teamOpenResponse, teamHighResponse] = await Promise.all([
        api.getTeamConversations(team.id, {
          page: 1,
          page_size: 1,
          filters: openFilter
        }),
        api.getTeamConversations(team.id, {
          page: 1,
          page_size: 1,
          priority: 'high',
          filters: openFilter
        })
      ])

      sidebarCounts.value[`team_${team.id}`] = teamOpenResponse.data.data.total || 0

      sidebarCounts.value[`team_high_${team.id}`] = teamHighResponse.data.data.total || 0
    }
  } catch (error) {
    console.error('Failed loading sidebar counts', error)
  } finally {
    loadingSidebarCounts = false
  }
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, handleInboxRefresh)
  window.addEventListener('focus', loadSidebarCounts)
  loadSidebarCounts()

  sidebarCountInterval = setInterval(() => {
    loadSidebarCounts()
  }, 5000)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, handleInboxRefresh)
  window.removeEventListener('focus', loadSidebarCounts)
  if (sidebarCountInterval) {
    clearInterval(sidebarCountInterval)
  }
})
</script>

<template>
  <SidebarProvider
    style="--sidebar-width: 14rem"
    :default-open="sidebarOpen"
    v-on:update:open="sidebarOpen = $event"
  >
    <!-- Contacts sidebar -->
    <template
      v-if="route.matched.some((record) => record.name && record.name.startsWith('contact'))"
    >
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.contact', 2) }}
                </span>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredContactsNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ t(item.allLabelKey) }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Reports sidebar -->
    <template
      v-if="
        userStore.hasReportTabPermissions &&
        route.matched.some((record) => record.name && record.name.startsWith('reports'))
      "
    >
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.report', 2) }}
                </span>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredReportsNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Admin Sidebar -->
    <template v-if="route.matched.some((record) => record.name && record.name.startsWith('admin'))">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex flex-col items-start justify-between w-full px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.admin') }}
                </span>

                <div class="text-xs text-muted-foreground">
                  ({{ settingsStore.settings['app.version'] }})
                </div>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredAdminNavItems" :key="item.titleKey">
                <SidebarMenuButton
                  v-if="!item.children"
                  :isActive="isActiveParent(item.href)"
                  asChild
                >
                  <router-link :to="item.href">
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>

                <Collapsible
                  v-else
                  class="group/collapsible"
                  :open="openAdminCollapsible === item.titleKey"
                  @update:open="toggleAdminCollapsible(item.titleKey)"
                >
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton :isActive="isActiveParent(item.href)">
                      <span>
                        {{ t(item.titleKey, item.isTitleKeyPlural === true ? 2 : 1) }}
                      </span>

                      <Badge
                        v-if="item.badge"
                        variant="outline"
                        class="ml-1.5 rounded-full uppercase tracking-[0.07em] font-medium text-[9px] leading-none px-[5.5px] py-[3px] bg-warning/10 text-warning-600 border-warning/50 shrink-0"
                      >
                        {{ item.badge }}
                      </Badge>

                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="child in item.children" :key="child.titleKey">
                        <SidebarMenuButton size="sm" :isActive="isActiveParent(child.href)" asChild>
                          <router-link :to="child.href">
                            <component :is="navIconMap[child.icon]" v-if="child.icon" />
                            <span>
                              {{ t(child.titleKey, child.isTitleKeyPlural === true ? 2 : 1) }}
                            </span>
                          </router-link>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </Collapsible>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Account sidebar -->
    <template v-if="isActiveParent('/account')">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.account') }}
                </span>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in accountNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>

                <SidebarMenuAction>
                  <span class="sr-only">{{ item.description }}</span>
                </SidebarMenuAction>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Inbox sidebar -->
    <template v-if="route.path && isInboxRoute(route.path)">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex items-center justify-between w-full px-1">
                <div class="font-semibold text-xl">
                  <span>{{ t('globals.terms.inbox') }}</span>
                </div>

                <div class="mr-1 mt-1 transition-colors">
                  <router-link :to="{ name: 'search' }">
                    <Search
                      size="18"
                      stroke-width="2.5"
                      class="text-muted-foreground hover:text-foreground"
                    />
                  </router-link>
                </div>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <MobileDrawerNav />

          <SidebarGroup>
            <SidebarMenu>
              <!-- New conversation -->
              <SidebarMenuItem>
                <SidebarMenuButton @click="emit('createConversation')">
                  <Plus />
                  <span>{{ t('conversation.newConversation') }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Meine Tickets -->
              <Collapsible class="group/collapsible" v-model:open="myTicketsOpen">
                <SidebarMenuItem>
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton :isActive="isActiveParent('/inboxes/assigned')">
                      <User />

                      <span>
                        {{ t('globals.terms.myInbox') }}
                      </span>

                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <!-- Hohe Priorität -->
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :isActive="
                            isActiveParent('/inboxes/assigned') &&
                            route.query.status === 'Open' &&
                            route.query.priority === 'high'
                          "
                          @click="
                            navigateToInbox('assigned', {
                              status: 'Open',
                              priority: 'high'
                            })
                          "
                        >
                          <span>Hohe Priorität</span>

                          <UnreadCountBadge :count="sidebarCounts.assigned_high || 0" />
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>

                      <!-- Offen -->
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :isActive="
                            isActiveParent('/inboxes/assigned') &&
                            route.query.status === 'Open' &&
                            !route.query.priority
                          "
                          @click="
                            navigateToInbox('assigned', {
                              status: 'Open'
                            })
                          "
                        >
                          <span>Offen</span>

                          <UnreadCountBadge :count="sidebarCounts.assigned || 0" />
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>

                      <!-- Weitere Status -->
                      <SidebarMenuSubItem
                        v-for="item in [
                          { label: 'Beantwortet', status: 'Resolved' },
                          { label: 'Schlummernd', status: 'Snoozed' },
                          { label: 'Geschlossen', status: 'Closed' }
                        ]"
                        :key="item.status"
                      >
                        <SidebarMenuButton
                          size="sm"
                          :isActive="
                            isActiveParent('/inboxes/assigned') &&
                            route.query.status === item.status
                          "
                          @click="
                            navigateToInbox('assigned', {
                              status: item.status
                            })
                          "
                        >
                          <span>{{ item.label }}</span>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Von mir erstellt -->
              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/created')"
                  @click="navigateToInbox('created')"
                >
                  <FilePlus />
                  <span>Von mir erstellt</span>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Sichtbar für mich -->
              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="
                    isCustomerSupport
                      ? isActiveParent('/inboxes/visible-internal')
                      : isActiveParent('/inboxes/visible')
                  "
                  @click="navigateToInbox(isCustomerSupport ? 'visible-internal' : 'visible')"
                >
                  <Eye />
                  <span>Sichtbar für mich</span>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Private Postfächer -->
              <Collapsible v-if="personalInboxes.length" v-model:open="personalInboxOpen" as-child>
                <SidebarMenuItem>
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton>
                      <Mail class="h-4 w-4" />
                      <span>{{ t('inbox.personalMailboxes') }}</span>
                      <ChevronRight
                        class="ml-auto transition-transform duration-200"
                        :class="{ 'rotate-90': personalInboxOpen }"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="inbox in personalInboxes" :key="inbox.id">
                        <Collapsible
                          :open="personalInboxOpenStates[inbox.id] !== false"
                          @update:open="personalInboxOpenStates[inbox.id] = $event"
                        >
                          <CollapsibleTrigger as-child>
                            <SidebarMenuButton size="sm">
                              <span>{{ inbox.name }}</span>
                              <ChevronRight
                                class="ml-auto transition-transform duration-200"
                                :class="{
                                  'rotate-90': personalInboxOpenStates[inbox.id] !== false
                                }"
                              />
                            </SidebarMenuButton>
                          </CollapsibleTrigger>
                          <CollapsibleContent>
                            <SidebarMenuSub>
                              <SidebarMenuSubItem
                                v-for="item in personalInboxItems"
                                :key="item.key"
                              >
                                <SidebarMenuButton
                                  size="sm"
                                  :isActive="isPersonalRouteActive(inbox.id, item.query)"
                                  @click="navigateToPersonalInbox(inbox.id, item.query)"
                                >
                                  <span>{{ item.label }}</span>
                                  <UnreadCountBadge
                                    v-if="item.count"
                                    :count="sidebarCounts[`personal_${inbox.id}_${item.key}`] || 0"
                                  />
                                </SidebarMenuButton>
                              </SidebarMenuSubItem>
                            </SidebarMenuSub>
                          </CollapsibleContent>
                        </Collapsible>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Team-Posteingänge -->
              <Collapsible
                v-if="userTeams.length"
                class="group/collapsible"
                v-model:open="teamInboxOpen"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton>
                      <UsersRound />

                      <span>
                        {{ t('globals.terms.teamInbox', 2) }}
                      </span>

                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="team in userTeams" :key="team.id">
                        <Collapsible
                          :open="isTeamInboxOpen(team.id)"
                          @update:open="setTeamInboxOpen(team.id, $event)"
                        >
                          <div class="w-full">
                            <CollapsibleTrigger as-child>
                              <SidebarMenuButton size="sm">
                                <div class="flex items-center gap-2">
                                  <span>{{ team.emoji }}</span>
                                  <span>{{ team.name }}</span>
                                </div>

                                <ChevronRight
                                  class="ml-auto transition-transform duration-200"
                                  :class="{
                                    'rotate-90': isTeamInboxOpen(team.id)
                                  }"
                                />
                              </SidebarMenuButton>
                            </CollapsibleTrigger>

                            <CollapsibleContent>
                              <SidebarMenuSub>
                                <!-- Team Hohe Priorität -->
                                <SidebarMenuSubItem>
                                  <SidebarMenuButton
                                    size="sm"
                                    :isActive="isTeamRouteActive(team.id, 'Open', 'high')"
                                    @click="
                                      navigateToTeamInbox(team.id, {
                                        status: 'Open',
                                        priority: 'high'
                                      })
                                    "
                                  >
                                    <span>Hohe Priorität</span>

                                    <UnreadCountBadge
                                      :count="sidebarCounts[`team_high_${team.id}`] || 0"
                                    />
                                  </SidebarMenuButton>
                                </SidebarMenuSubItem>

                                <!-- Team Offen -->
                                <SidebarMenuSubItem>
                                  <SidebarMenuButton
                                    size="sm"
                                    :isActive="isTeamRouteActive(team.id, 'Open')"
                                    @click="
                                      navigateToTeamInbox(team.id, {
                                        status: 'Open'
                                      })
                                    "
                                  >
                                    <span>Offen</span>

                                    <UnreadCountBadge
                                      :count="sidebarCounts[`team_${team.id}`] || 0"
                                    />
                                  </SidebarMenuButton>
                                </SidebarMenuSubItem>

                                <!-- Team weitere Status -->
                                <SidebarMenuSubItem
                                  v-for="item in [
                                    {
                                      label: 'Beantwortet',
                                      status: 'Resolved'
                                    },
                                    {
                                      label: 'Schlummernd',
                                      status: 'Snoozed'
                                    },
                                    {
                                      label: 'Geschlossen',
                                      status: 'Closed'
                                    }
                                  ]"
                                  :key="item.status"
                                >
                                  <SidebarMenuButton
                                    size="sm"
                                    :isActive="isTeamRouteActive(team.id, item.status)"
                                    @click="
                                      navigateToTeamInbox(team.id, {
                                        status: item.status
                                      })
                                    "
                                  >
                                    <span>{{ item.label }}</span>
                                  </SidebarMenuButton>
                                </SidebarMenuSubItem>
                              </SidebarMenuSub>
                            </CollapsibleContent>
                          </div>
                        </Collapsible>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Kundentickets -->
              <Collapsible
                v-if="isCustomerSupport || isAdmin"
                class="group/collapsible"
                v-model:open="customerTicketsOpen"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton :isActive="isActiveParent('/inboxes/customer')">
                      <Mail />

                      <span>Kundentickets</span>

                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <!-- Hohe Priorität -->
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :isActive="isActiveParent('/inboxes/customer-high')"
                          @click="
                            navigateToInbox('customer-high', {
                              status: 'Open'
                            })
                          "
                        >
                          <span>Hohe Priorität</span>

                          <UnreadCountBadge :count="sidebarCounts.customer_high || 0" />
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>

                      <!-- Offen -->
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :isActive="
                            isActiveParent('/inboxes/customer') &&
                            !route.path.startsWith('/inboxes/customer-high') &&
                            route.query.status === 'Open' &&
                            !route.query.filters
                          "
                          @click="
                            navigateToInbox('customer', {
                              status: 'Open'
                            })
                          "
                        >
                          <span>Offen</span>

                          <UnreadCountBadge :count="sidebarCounts.customer || 0" />
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>

                      <!-- Weitere Status -->
                      <SidebarMenuSubItem
                        v-for="item in [
                          { label: 'Beantwortet', status: 'Resolved' },
                          { label: 'Schlummernd', status: 'Snoozed' },
                          { label: 'Geschlossen', status: 'Closed' }
                        ]"
                        :key="item.status"
                      >
                        <SidebarMenuButton
                          size="sm"
                          :isActive="
                            isActiveParent('/inboxes/customer') &&
                            !route.path.startsWith('/inboxes/customer-high') &&
                            route.query.status === item.status &&
                            !route.query.filters
                          "
                          @click="
                            navigateToInbox('customer', {
                              status: item.status
                            })
                          "
                        >
                          <span>{{ item.label }}</span>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>

                      <!-- Spam -->
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :isActive="
                            isActiveParent('/inboxes/customer') &&
                            route.query.filters === spamFilter
                          "
                          @click="
                            navigateToInbox('customer', {
                              filters: spamFilter
                            })
                          "
                        >
                          <span>Spam</span>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Service-Mails -->
              <SidebarMenuItem v-if="isCustomerSupport || isAdmin">
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/service-mails')"
                  @click="navigateToInbox('service-mails')"
                >
                  <MailCheck />
                  <span>Service-Mails</span>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Nicht zugewiesen - Admin only -->
              <SidebarMenuItem v-if="isAdmin">
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/unassigned')"
                  @click="navigateToInbox('unassigned')"
                >
                  <CircleDashed />

                  <div class="flex items-center justify-between w-full">
                    <span>
                      {{ t('globals.terms.unassigned') }}
                    </span>

                    <UnreadCountBadge :count="sidebarCounts.unassigned || 0" />
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Alle - Admin only -->
              <SidebarMenuItem v-if="isAdmin">
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/all')"
                  @click="navigateToInbox('all')"
                >
                  <List />

                  <span>
                    {{ t('globals.messages.all') }}
                  </span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Main Content Area -->
    <SidebarInset class="bg-canvas !min-h-0 !h-full">
      <slot></slot>
    </SidebarInset>
  </SidebarProvider>
</template>

<style scoped>
:deep(.sidebar-secondary) {
  @apply border border-sidebar-border ml-[3.2rem] rounded-lg overflow-hidden;
  top: 0.4rem !important;
  bottom: 0.35rem !important;
  height: auto !important;
}

/* Override SidebarProvider height */
:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}
</style>