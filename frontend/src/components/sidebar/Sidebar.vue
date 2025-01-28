<script setup>
import { RouterLink, useRoute } from 'vue-router'
import SidebarNavUser from './SidebarNavUser.vue'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarSeparator,
  SidebarGroupContent,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuSubButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarRail
} from '@/components/ui/sidebar'
import {
  Users,
  Inbox,
  ChevronRight,
  SlidersHorizontal,
  Shield,
  FileLineChart,
  EllipsisVertical,
  Plus,
  Search,
  MessageCircle
} from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { useConversationStore } from '@/stores/conversation'
import ConversationSideBar from '@/components/conversation/sidebar/ConversationSideBar.vue'

const route = useRoute()

defineProps({
  isLoading: Boolean,
  open: Boolean,
  activeItem: { type: Object, default: () => {} },
  activeGroup: { type: Object, default: () => {} },
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] }
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
    .filter((item) => {
      // Check if the item has permissions and if all are satisfied
      const hasPermission =
        !item.permissions || item.permissions.every((permission) => userStore.can(permission))

      // Check if the children have permissions
      const filteredChildren = item.children
        ? item.children.filter(
            (child) =>
              !child.permissions ||
              child.permissions.every((permission) => userStore.can(permission))
          )
        : []

      // Include item only if it has permission and either no children or children with permission
      return hasPermission && (filteredChildren.length > 0 || !item.children)
    })
    .map((item) => ({
      ...item,
      children: item.children
        ? item.children.filter(
            (child) =>
              !child.permissions ||
              child.permissions.every((permission) => userStore.can(permission))
          )
        : []
    }))
}

const filteredAdminNavItems = computed(() => filterNavItemsByPermissions(adminNavItems, userStore))

const hasAdminAccess = computed(() => {
  return filterNavItemsByPermissions(adminNavItems, userStore).length > 0
})

const filterReportsNavItemsByPermissions = (navItems, userStore) => {
  return navItems.filter(
    (item) => !item.permissions || item.permissions.every((permission) => userStore.can(permission))
  )
}

const filteredReportsNavItems = computed(() =>
  filterReportsNavItemsByPermissions(reportsNavItems, userStore)
)

const hasReportsAccess = computed(() => filteredReportsNavItems.value.length > 0)

const reportsNavItems = [
  {
    title: 'Overview',
    href: '/reports/overview',
    permissions: ['reports:manage']
  }
]

const adminNavItems = [
  {
    title: 'Workspace',
    children: [
      {
        title: 'General',
        href: '/admin/general',
        permissions: ['general_settings:manage']
      },
      {
        title: 'Business Hours',
        href: '/admin/business-hours',
        permissions: ['business_hours:manage']
      },
      {
        title: 'SLA',
        href: '/admin/sla',
        permissions: ['sla:manage']
      }
    ]
  },
  {
    title: 'Conversations',
    description: 'Manage tags, macros, and statuses.',
    children: [
      {
        title: 'Tags',
        href: '/admin/conversations/tags',
        permissions: ['tags:manage']
      },
      {
        title: 'Macros',
        href: '/admin/conversations/macros',
        permissions: ['macros:manage']
      },
      {
        title: 'Statuses',
        href: '/admin/conversations/statuses',
        permissions: ['status:manage']
      }
    ]
  },
  {
    title: 'Inboxes',
    description: 'Manage inboxes.',
    children: [
      {
        title: 'Inboxes',
        href: '/admin/inboxes',
        permissions: ['inboxes:manage']
      }
    ]
  },
  {
    title: 'Teammates',
    description: 'Manage users, teams, and roles.',
    children: [
      {
        title: 'Users',
        href: '/admin/teams/users',
        permissions: ['users:manage']
      },
      {
        title: 'Teams',
        href: '/admin/teams/teams',
        permissions: ['teams:manage']
      },
      {
        title: 'Roles',
        href: '/admin/teams/roles',
        permissions: ['roles:manage']
      }
    ]
  },
  {
    title: 'Automations',
    description: 'Manage automation rules.',
    children: [
      {
        title: 'Automations',
        href: '/admin/automations',
        permissions: ['automations:manage']
      }
    ]
  },
  {
    title: 'Notifications',
    description: 'Manage email notifications.',
    children: [
      {
        title: 'Notifications',
        href: '/admin/notification',
        permissions: ['notification_settings:manage']
      }
    ]
  },
  {
    title: 'Templates',
    description: 'Manage email templates.',
    children: [
      {
        title: 'Templates',
        href: '/admin/templates',
        permissions: ['templates:manage']
      }
    ]
  },
  {
    title: 'Security',
    description: 'Configure SSO and security.',
    children: [
      {
        title: 'OpenID Connect SSO',
        href: '/admin/oidc',
        permissions: ['oidc:manage']
      }
    ]
  }
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
  return conversationStore.current || conversationStore.conversation.loading
})
</script>

<template>
  <div class="flex flex-row justify-between h-full">
    <div class="flex-1">
      <SidebarProvider
        :open="open"
        @update:open="($event) => emit('update:open', $event)"
        style="--sidebar-width: 16rem"
      >
        <!-- Flex Container that holds all the sidebar components -->
        <Sidebar
          collapsible="icon"
          class="overflow-hidden [&>[data-sidebar=sidebar]]:flex-row !border-r-0"
        >
          <!-- Left Sidebar (Icon Sidebar) -->
          <Sidebar collapsible="none" class="!w-[calc(var(--sidebar-width-icon)_+_1px)] border-r">
            <SidebarHeader>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    :isActive="isActiveParent('#')"
                    size="sm"
                    asChild
                    class="md:h-8 md:p-0"
                  >
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
                      <SidebarMenuButton
                        :isActive="route.path && route.path.startsWith('/admin')"
                        asChild
                      >
                        <router-link :to="{ name: 'admin' }">
                          <Shield class="w-5 h-5" />
                        </router-link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                    <SidebarMenuItem v-if="hasReportsAccess">
                      <SidebarMenuButton
                        :isActive="route.path && route.path.startsWith('/reports')"
                        asChild
                      >
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
            v-if="
              hasReportsAccess &&
              route.matched.some((record) => record.name && record.name.startsWith('reports'))
            "
          >
            <Sidebar collapsible="none" class="!border-r-0 bg-white">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      :isActive="isActiveParent('/reports/overview')"
                      asChild
                      size="md"
                    >
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
          <template
            v-if="route.matched.some((record) => record.name && record.name.startsWith('admin'))"
          >
            <Sidebar collapsible="none" class="!border-r-0 bg-white">
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
                      <SidebarMenuButton
                        v-if="!item.children"
                        :isActive="isActiveParent(item.href)"
                        asChild
                      >
                        <router-link :to="item.href">
                          <span>{{ item.title }}</span>
                        </router-link>
                      </SidebarMenuButton>

                      <Collapsible
                        v-else
                        class="group/collapsible"
                        :default-open="isActiveParent(item.href)"
                      >
                        <CollapsibleTrigger as-child>
                          <SidebarMenuButton :isActive="isActiveParent(item.href)">
                            <span>{{ item.title }}</span>
                            <ChevronRight
                              class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                            />
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem v-for="child in item.children" :key="child.title">
                              <SidebarMenuSubButton
                                size="sm"
                                :isActive="isActiveParent(child.href)"
                                asChild
                              >
                                <router-link :to="child.href">
                                  <span>{{ child.title }}</span>
                                </router-link>
                              </SidebarMenuSubButton>
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
            <Sidebar collapsible="none" class="!border-r-0 bg-white">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      :isActive="isActiveParent('/account/profile')"
                      asChild
                      size="md"
                    >
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
            <Sidebar collapsible="none" class="!border-r-0 bg-white">
              <SidebarHeader>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton asChild>
                      <div class="flex items-center justify-between w-full">
                        <div class="font-semibold text-2xl">Inbox</div>
                        <div class="ml-auto">
                          <router-link :to="{ name: 'search' }">
                            <Search
                              class="transition-transform duration-200 hover:scale-110 cursor-pointer"
                              size="20"
                            />
                          </router-link>
                        </div>
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
                              class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                            />
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                size="sm"
                                :isActive="isActiveParent('/inboxes/assigned')"
                                asChild
                              >
                                <router-link :to="{ name: 'inbox', params: { type: 'assigned' } }">
                                  <span>My inbox</span>
                                </router-link>
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                size="sm"
                                :isActive="isActiveParent('/inboxes/unassigned')"
                                asChild
                              >
                                <router-link
                                  :to="{ name: 'inbox', params: { type: 'unassigned' } }"
                                >
                                  <span>Unassigned</span>
                                </router-link>
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                size="sm"
                                :isActive="isActiveParent('/inboxes/all')"
                                asChild
                              >
                                <router-link :to="{ name: 'inbox', params: { type: 'all' } }">
                                  <span>All</span>
                                </router-link>
                              </SidebarMenuSubButton>
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
                                class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                              />
                            </router-link>
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub v-for="team in userTeams" :key="team.id">
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                size="sm"
                                :isActive="isActiveParent(`/teams/${team.id}`)"
                                asChild
                              >
                                <router-link
                                  :to="{ name: 'team-inbox', params: { teamID: team.id } }"
                                >
                                  {{ team.emoji }}<span>{{ team.name }}</span>
                                </router-link>
                              </SidebarMenuSubButton>
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
                                <Plus
                                  size="18"
                                  @click.stop="openCreateViewDialog"
                                  class="rounded-lg cursor-pointer opacity-0 transition-all duration-200 group-hover:opacity-100 hover:bg-gray-200 hover:shadow-sm text-gray-600 hover:text-gray-800 transform hover:scale-105 active:scale-100 p-1"
                                />
                              </div>
                            </router-link>
                          </SidebarMenuButton>
                        </CollapsibleTrigger>

                        <SidebarMenuAction>
                          <ChevronRight
                            class="transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                            v-if="userViews.length"
                          />
                        </SidebarMenuAction>

                        <CollapsibleContent>
                          <SidebarMenuSub v-for="view in userViews" :key="view.id">
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                size="sm"
                                :isActive="isActiveParent(`/views/${view.id}`)"
                                asChild
                              >
                                <router-link
                                  :to="{ name: 'view-inbox', params: { viewID: view.id } }"
                                >
                                  <span>{{ view.name }}</span>
                                </router-link>
                              </SidebarMenuSubButton>

                              <SidebarMenuAction>
                                <DropdownMenu>
                                  <DropdownMenuTrigger asChild>
                                    <EllipsisVertical />
                                  </DropdownMenuTrigger>
                                  <DropdownMenuContent>
                                    <DropdownMenuItem @click="() => editView(view)">
                                      <span>Edit</span>
                                    </DropdownMenuItem>
                                    <DropdownMenuItem @click="() => deleteView(view)">
                                      <span>Delete</span>
                                    </DropdownMenuItem>
                                  </DropdownMenuContent>
                                </DropdownMenu>
                              </SidebarMenuAction>
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
      <SidebarProvider :open="true" style="--sidebar-width: 20rem">
        <Sidebar collapsible="none" side="right">
          <SidebarSeparator />
          <SidebarContent>
            <SidebarGroup style="padding: 0">
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
