<script setup>
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import api from '@/api'
import { useStorage } from '@vueuse/core'
import { RouterLink, useRoute } from 'vue-router'
import SidebarNavUser from './SidebarNavUser.vue'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenuBadge,
  SidebarHeader,
  SidebarInset,
  SidebarMenuSkeleton,
  SidebarMenu,
  SidebarSeparator,
  SidebarGroupContent,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarTrigger,
  SidebarRail,
} from '@/components/ui/sidebar'
import {
  AudioWaveform,
  BadgeCheck,
  Bell,
  Users,
  Bot,
  Inbox,
  ChevronRight,
  ChevronsUpDown,
  Command,
  CreditCard,
  SlidersHorizontal,
  Folder,
  Shield,
  FileLineChart,
  EllipsisVertical,
  MessageCircleHeart,
  Plus,
  MessageCircle,
} from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'
import { ref, onMounted, reactive, computed } from 'vue'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useConversationStore } from '@/stores/conversation'

import ConversationSideBar from '@/components/conversation/sidebar/ConversationSideBar.vue'

const route = useRoute()

defineProps({
  isLoading: Boolean,
  open: Boolean,
  activeItem: { type: Object, default: () => { } },
  activeGroup: { type: Object, default: () => { } },
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] },
})

const conversationStore = useConversationStore()
const userStore = useUserStore()

const emit = defineEmits(['createView', 'editView', 'deleteView', 'update:open'])

const openCreateViewDialog = () => {
  emit('createView')
}

const editView = (view) => {
  emit('editView', view)
}

const deleteView = (view) => {
  emit('deleteView', view)
}

const filterNavItemsByPermissions = (navItems, userStore) => {
  return navItems
    .filter(item => {
      // Check if the item has permissions and if all are satisfied
      const hasPermission = !item.permissions || item.permissions.every(permission => userStore.can(permission))

      // Check if the children have permissions
      const filteredChildren = item.children
        ? item.children.filter(child =>
          !child.permissions || child.permissions.every(permission => userStore.can(permission))
        )
        : []

      // Include item only if it has permission and either no children or children with permission
      return hasPermission && (filteredChildren.length > 0 || !item.children)
    })
    .map(item => ({
      ...item,
      children: item.children
        ? item.children.filter(child =>
          !child.permissions || child.permissions.every(permission => userStore.can(permission))
        )
        : []
    }))
}

const filteredAdminNavItems = computed(() => filterNavItemsByPermissions(adminNavItems, userStore))

const hasAdminAccess = computed(() => {
  return filterNavItemsByPermissions(adminNavItems, userStore).length > 0
})

const filterReportsNavItemsByPermissions = (navItems, userStore) => {
  return navItems.filter(item =>
    !item.permissions || item.permissions.every(permission => userStore.can(permission))
  )
}

const filteredReportsNavItems = computed(() =>
  filterReportsNavItemsByPermissions(reportsNavItems, userStore)
)

const hasReportsAccess = computed(() =>
  filteredReportsNavItems.value.length > 0
)


const reportsNavItems = [
  {
    title: 'Overview',
    href: '/reports/overview',
    permissions: ['reports:manage'],
  },
]


const adminNavItems = [
  {
    title: 'General',
    description: 'Configure general app settings',
    href: '/admin/general',
    children: [
      {
        title: 'General',
        href: '/admin/general',
        description: 'Configure general app settings',
        permissions: ['general_settings:manage'],
      }
    ],
  },
  {
    title: 'Conversations',
    href: '/admin/conversations',
    description: 'Manage tags, macros and statuses.',
    children: [
      {
        title: 'Tags',
        href: '/admin/conversations/tags',
        description: 'Manage conversation tags.',
        permissions: ['tags:manage'],
      },
      {
        title: 'Macros',
        href: '/admin/conversations/macros',
        description: 'Manage macros.',
        permissions: ['tags:manage'],
      },
      {
        title: 'Statuses',
        href: '/admin/conversations/statuses',
        description: 'Manage conversation statuses.',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'Inboxes',
    href: '/admin/inboxes',
    description: 'Manage your inboxes',
    children: [
      {
        title: 'Inboxes',
        href: '/admin/inboxes',
        description: 'Manage your inboxes',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'Teams',
    href: '/admin/teams',
    description: 'Manage teams, manage agents and roles',
    children: [
      {
        title: 'Users',
        href: '/admin/teams/users',
        description: 'Manage users',
        permissions: ['tags:manage'],
      },
      {
        title: 'Teams',
        href: '/admin/teams/teams',
        description: 'Manage teams',
        permissions: ['tags:manage'],
      },
      {
        title: 'Roles',
        href: '/admin/teams/roles',
        description: 'Manage roles',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'Automations',
    href: '/admin/automations',
    description: 'Manage automations and time triggers',
    children: [
      {
        title: 'Automations',
        href: '/admin/automations',
        description: 'Manage automations',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'Email notifications',
    href: '/admin/notification',
    description: 'Configure SMTP',
    children: [
      {
        title: 'Email notifications',
        href: '/admin/notification',
        description: 'Configure SMTP',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'Email templates',
    href: '/admin/templates',
    description: 'Manage email templates',
    children: [
      {
        title: 'Email templates',
        href: '/admin/templates',
        description: 'Manage email templates',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'Business hours',
    href: '/admin/business-hours',
    description: 'Manage business hours',
    children: [
      {
        title: 'Business hours',
        href: '/admin/business-hours',
        description: 'Manage business hours',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'SLA',
    href: '/admin/sla',
    description: 'Manage SLA policies',
    children: [
      {
        title: 'SLA',
        href: '/admin/sla',
        description: 'Manage SLA policies',
        permissions: ['tags:manage'],
      },
    ],
  },
  {
    title: 'SSO',
    href: '/admin/oidc',
    description: 'Manage OpenID SSO configurations',
    children: [
      {
        title: 'SSO',
        href: '/admin/oidc',
        description: 'Manage OpenID SSO configurations',
        permissions: ['tags:manage'],
      },
    ],
  },
]

const accountNavItems = [
  {
    title: 'Profile',
    href: '/account/profile',
    description: 'Update your profile'
  }
]

const isActiveParent = (parentHref) => {
  return route.path.startsWith(parentHref)
}

const isInboxRoute = (path) => {
  return path.startsWith('/inboxes') || path.startsWith('/teams') || path.startsWith('/views')
}

const hasConversationOpen = computed(() => {
  return conversationStore.current
})
</script>

<template>
  <div class="flex flex-row justify-between h-full">
    <div class="flex-1">
      <SidebarProvider :open="open" @update:open="($event) => emit('update:open', $event)" style="--sidebar-width: 16rem;">
        <!-- Flex Container that holds all the sidebar components -->
        <Sidebar collapsible="icon" class="overflow-hidden [&>[data-sidebar=sidebar]]:flex-row !border-r-0">

          <!-- Left Sidebar (Icon Sidebar) -->
          <Sidebar collapsible="none" class="!w-[calc(var(--sidebar-width-icon)_+_1px)] border-r">
            <SidebarHeader>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton :isActive="isActiveParent('#')" size="sm" asChild class="md:h-8 md:p-0">
                    <a href="#">
                      <div class="flex items-center justify-center w-full h-full">
                        <MessageCircle size="25" />
                      </div>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarHeader>
            <SidebarContent>
              <SidebarGroup>
                <SidebarGroupContent class="px-1.5 md:px-0">
                  <SidebarMenu>
                    <SidebarMenuItem>
                      <SidebarMenuButton :isActive="route.path && isInboxRoute(route.path)" asChild>
                        <router-link :to="{ name: 'inboxes' }">
                          <Inbox class="w-5 h-5" />
                        </router-link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                    <SidebarMenuItem v-if="hasAdminAccess">
                      <SidebarMenuButton :isActive="route.path && route.path.startsWith('/admin')" asChild>
                        <router-link :to="{ name: 'admin' }">
                          <Shield class="w-5 h-5" />
                        </router-link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                    <SidebarMenuItem v-if="hasReportsAccess">
                      <SidebarMenuButton :isActive="route.path && route.path.startsWith('/reports')" asChild>
                        <router-link :to="{ name: 'reports' }">
                          <FileLineChart class="w-5 h-5" />
                        </router-link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            </SidebarContent>
            <SidebarFooter>
              <SidebarNavUser />
            </SidebarFooter>
          </Sidebar>

          <!-- Reports sidebar -->
          <template
            v-if="hasReportsAccess && route.matched.some(record => record.name && record.name.startsWith('reports'))">
            <Sidebar collapsible="none" class="!border-r-0 bg-white ">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton :isActive="isActiveParent('/reports/overview')" asChild size="md">
                      <div>
                        <span class="font-semibold text-2xl">Reports</span>
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarHeader>
              <SidebarSeparator />
              <SidebarContent>
                <SidebarGroup>
                  <SidebarMenu>
                    <SidebarMenuItem v-for="item in filteredReportsNavItems" :key="item.title">
                      <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                        <router-link :to="item.href">
                          <span>{{ item.title }}</span>
                        </router-link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  </SidebarMenu>
                </SidebarGroup>
              </SidebarContent>
              <SidebarRail />
            </Sidebar>
          </template>


          <!-- Admin Sidebar -->
          <template v-if="route.matched.some(record => record.name && record.name.startsWith('admin'))">
            <Sidebar collapsible="none" class="!border-r-0 bg-white ">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton :isActive="isActiveParent('/admin')" asChild size="md">
                      <div>
                        <span class="font-semibold text-2xl">Admin</span>
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarHeader>
              <SidebarSeparator />
              <SidebarContent>
                <SidebarGroup>
                  <SidebarMenu>
                    <SidebarMenuItem v-for="item in filteredAdminNavItems" :key="item.title">
                      <SidebarMenuButton v-if="!item.children" :isActive="isActiveParent(item.href)" asChild>
                        <router-link :to="item.href">
                          <span>{{ item.title }}</span>
                        </router-link>
                      </SidebarMenuButton>

                      <Collapsible v-else class="group/collapsible" :default-open="isActiveParent(item.href)">
                        <CollapsibleTrigger as-child>
                          <SidebarMenuButton :isActive="isActiveParent(item.href)">
                            <span>{{ item.title }}</span>
                            <ChevronRight
                              class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem v-for="child in item.children" :key="child.title">
                              <SidebarMenuButton :isActive="isActiveParent(child.href)" asChild>
                                <router-link :to="child.href">
                                  <span>{{ child.title }}</span>
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
              <SidebarRail />
            </Sidebar>
          </template>

          <!-- Account sidebar -->
          <template v-if="isActiveParent('/account')">
            <Sidebar collapsible="none" class="!border-r-0 bg-white ">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton :isActive="isActiveParent('/account/profile')" asChild size="md">
                      <div>
                        <span class="font-semibold text-2xl">Account</span>
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarHeader>
              <SidebarSeparator />
              <SidebarContent>
                <SidebarGroup>
                  <SidebarMenu>
                    <SidebarMenuItem v-for="item in accountNavItems" :key="item.title">
                      <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                        <router-link :to="item.href">
                          <span>{{ item.title }}</span>
                        </router-link>
                      </SidebarMenuButton>
                      <SidebarMenuAction>
                        <span class="sr-only">{{ item.description }}</span>
                      </SidebarMenuAction>
                    </SidebarMenuItem>
                  </SidebarMenu>
                </SidebarGroup>
              </SidebarContent>
              <SidebarRail />
            </Sidebar>
          </template>

          <!-- Conversation Sidebar -->
          <template v-if="route.path && isInboxRoute(route.path)">
            <Sidebar collapsible="none" class="!border-r-0 bg-white ">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton asChild>
                      <div>
                        <span class="font-semibold text-2xl">Inbox</span>
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarHeader>
              <SidebarSeparator />
              <SidebarContent>

                <SidebarGroup>
                  <SidebarMenu>
                    <!-- Inboxes Collapsible -->
                    <Collapsible class="group/collapsible" defaultOpen>
                      <SidebarMenuItem>
                        <CollapsibleTrigger as-child>
                          <SidebarMenuButton>
                            <MessageCircle />
                            <span>Inboxes</span>
                            <ChevronRight
                              class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem>
                              <SidebarMenuButton :isActive="isActiveParent('/inboxes/assigned')" asChild>
                                <router-link :to="{ name: 'inbox', params: { type: 'assigned' } }">
                                  <span>My inbox</span>
                                </router-link>
                              </SidebarMenuButton>
                            </SidebarMenuSubItem>
                            <SidebarMenuSubItem>
                              <SidebarMenuButton :isActive="isActiveParent('/inboxes/unassigned')" asChild>
                                <router-link :to="{ name: 'inbox', params: { type: 'unassigned' } }">
                                  <span>Unassigned</span>
                                </router-link>
                              </SidebarMenuButton>
                            </SidebarMenuSubItem>
                            <SidebarMenuSubItem>
                              <SidebarMenuButton :isActive="isActiveParent('/inboxes/all')" asChild>
                                <router-link :to="{ name: 'inbox', params: { type: 'all' } }">
                                  <span>All</span>
                                </router-link>
                              </SidebarMenuButton>
                            </SidebarMenuSubItem>
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </SidebarMenuItem>
                    </Collapsible>

                    <!-- Team Inboxes -->
                    <Collapsible defaultOpen class="group/collapsible" v-if="userTeams.length">
                      <SidebarMenuItem>
                        <CollapsibleTrigger as-child>
                          <SidebarMenuButton asChild>
                            <router-link to="#">
                              <Users />
                              <span>Team inboxes</span>
                              <ChevronRight
                                class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                            </router-link>
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem v-for="team in userTeams" :key="team.id">
                              <SidebarMenuButton :isActive="isActiveParent(`/teams/${team.id}`)" asChild>
                                <router-link :to="{ name: 'team-inbox', params: { teamID: team.id } }">
                                  {{ team.emoji }}<span>{{ team.name }}</span>
                                </router-link>
                              </SidebarMenuButton>
                            </SidebarMenuSubItem>
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </SidebarMenuItem>
                    </Collapsible>

                    <!-- Views -->
                    <Collapsible class="group/collapsible" defaultOpen>
                      <SidebarMenuItem>
                        <CollapsibleTrigger as-child>
                          <SidebarMenuButton asChild>
                            <router-link to="#">
                              <SlidersHorizontal />
                              <span>Views</span>
                              <div>
                                <Plus size="18" @click.stop="openCreateViewDialog" class="rounded-lg cursor-pointer opacity-0 transition-all duration-200 
                            group-hover:opacity-100 hover:bg-gray-200 hover:shadow-sm
                            text-gray-600 hover:text-gray-800 transform hover:scale-105 
                            active:scale-100 p-1" />
                              </div>
                            </router-link>
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <SidebarMenuAction>
                          <ChevronRight
                            class="transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                            v-if="userViews.length" />
                        </SidebarMenuAction>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem v-for="view in userViews" :key="view.id">
                              <SidebarMenuButton :isActive="isActiveParent(`/views/${view.id}`)" asChild>
                                <router-link :to="{ name: 'view-inbox', params: { viewID: view.id } }">
                                  <span>{{ view.name }}</span>
                                </router-link>
                              </SidebarMenuButton>
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <SidebarMenuAction>
                                    <EllipsisVertical />
                                  </SidebarMenuAction>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent side="right">
                                  <DropdownMenuItem @click="() => editView(view)">
                                    <span>Edit</span>
                                  </DropdownMenuItem>
                                  <DropdownMenuItem @click="() => deleteView(view)">
                                    <span>Delete</span>
                                  </DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </SidebarMenuSubItem>
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </SidebarMenuItem>
                    </Collapsible>
                  </SidebarMenu>
                </SidebarGroup>
              </SidebarContent>
            </Sidebar>
          </template>
        </Sidebar>

        <!-- Main Content Area -->
        <SidebarInset>
          <slot></slot>
        </SidebarInset>
      </SidebarProvider>

    </div>

    <!-- Right Sidebar with conversation details -->
    <div v-if="hasConversationOpen">
      <SidebarProvider :open="true" style="--sidebar-width: 20rem;">
        <Sidebar collapsible="none" side="right">
          <SidebarSeparator />
          <SidebarContent>
            <SidebarGroup style="padding: 0;">
              <SidebarMenu>
                <SidebarMenuItem>
                  <ConversationSideBar />
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroup>
          </SidebarContent>
          <SidebarRail />
        </Sidebar>
      </SidebarProvider>
    </div>
  </div>
</template>
