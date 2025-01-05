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

const route = useRoute()

defineProps({
  isLoading: Boolean,
  open: Boolean,
  activeItem: { type: Object, default: () => { } },
  activeGroup: { type: Object, default: () => { } },
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] },
})

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

const reportsNavItems = [
  {
    title: 'Overview',
    href: '/reports/dashboard',
  },
]

const adminNavItems = [
  {
    title: 'General',
    description: 'Configure general app settings',
    children: [
      {
        title: 'General',
        href: '/admin/general',
        description: 'Configure general app settings',
      }
    ],
  },
  {
    title: 'Conversations',
    href: '/admin/conversations',
    description: 'Manage tags, canned responses and statuses.',
    children: [
      {
        title: 'Tags',
        href: '/admin/conversations/tags',
        description: 'Manage conversation tags.',
      },
      {
        title: 'Canned responses',
        href: '/admin/conversations/canned-responses',
        description: 'Manage canned responses.',
      },
      {
        title: 'Statuses',
        href: '/admin/conversations/statuses',
        description: 'Manage conversation statuses.',
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
      },
      {
        title: 'Teams',
        href: '/admin/teams/teams',
        description: 'Manage teams',
      },
      {
        title: 'Roles',
        href: '/admin/teams/roles',
        description: 'Manage roles',
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
      },
    ],
  },
  {
    title: 'OpenID Connect SSO',
    href: '/admin/oidc',
    description: 'Manage OpenID SSO configurations',
    children: [
      {
        title: 'OpenID Connect SSO',
        href: '/admin/oidc',
        description: 'Manage OpenID SSO configurations',
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
</script>

<template>
  <SidebarProvider :open="open" @update:open="($event) => emit('update:open', $event)">
    <!-- Flex Container to Hold Both Sidebars -->
    <Sidebar collapsible="icon" class="overflow-hidden [&>[data-sidebar=sidebar]]:flex-row !border-r-0">

      <!-- Left Sidebar (Icon Sidebar) -->
      <Sidebar collapsible="none" class="!w-[calc(var(--sidebar-width-icon)_+_1px)] border-r">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="sm" asChild class="md:h-8 md:p-0">
                <a href="#">
                  <div class="flex items-center justify-center w-full h-full">
                    <MessageCircleHeart class="w-6 h-6" />
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
                  <router-link :to="{ name: 'conversations-list', params: { type: CONVERSATION_LIST_TYPE.ASSIGNED } }">
                    <SidebarMenuButton>
                      <Inbox class="w-5 h-5" />
                    </SidebarMenuButton>
                  </router-link>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <router-link :to="{ name: 'admin' }">
                    <SidebarMenuButton>
                      <Shield class="w-5 h-5" />
                    </SidebarMenuButton>
                  </router-link>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <router-link :to="{ name: 'dashboard' }">
                    <SidebarMenuButton>
                      <FileLineChart class="w-5 h-5" />
                    </SidebarMenuButton>
                  </router-link>
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
      <template v-if="route.matched.some(record => record.name && record.name.startsWith('reports'))">
        <Sidebar collapsible="none" class="!border-r-0 bg-white ">
          <SidebarHeader>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild size="md">
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
                <SidebarMenuItem v-for="item in reportsNavItems" :key="item.title">
                  <Collapsible class="group/collapsible" v-if="item.children && item.children.length">
                    <CollapsibleTrigger as-child>
                      <SidebarMenuButton>
                        <span>{{ item.title }}</span>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                      </SidebarMenuButton>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        <SidebarMenuSubItem v-for="child in item.children" :key="child.title">
                          <SidebarMenuButton asChild>
                            <router-link :to="child.href">
                              <span>{{ child.title }}</span>
                            </router-link>
                          </SidebarMenuButton>
                        </SidebarMenuSubItem>
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </Collapsible>
                  <SidebarMenuButton v-else asChild>
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

      <!-- Admin Sidebar -->
      <template v-if="route.matched.some(record => record.name && record.name.startsWith('admin'))">
        <Sidebar collapsible="none" class="!border-r-0 bg-white ">
          <SidebarHeader>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild size="md">
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
                <SidebarMenuItem v-for="item in adminNavItems" :key="item.title">
                  <Collapsible class="group/collapsible" v-if="item.children && item.children.length">
                    <CollapsibleTrigger as-child>
                      <SidebarMenuButton>
                        <span>{{ item.title }}</span>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                      </SidebarMenuButton>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        <SidebarMenuSubItem v-for="child in item.children" :key="child.title">
                          <SidebarMenuButton asChild>
                            <router-link :to="child.href">
                              <span>{{ child.title }}</span>
                            </router-link>
                          </SidebarMenuButton>
                        </SidebarMenuSubItem>
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </Collapsible>
                  <SidebarMenuButton v-else asChild>
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

      <!-- Account sidebar -->
      <template v-if="route.matched.some(record => record.name && record.name.startsWith('account'))">
        <Sidebar collapsible="none" class="!border-r-0 bg-white ">
          <SidebarHeader>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild size="md">
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
                  <SidebarMenuButton asChild>
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
      <template v-if="route.name && route.name.startsWith('conversations')">

        <Sidebar collapsible="none" class="!border-r-0 bg-white ">
          <SidebarHeader>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild size="md">
                  <div>
                    <span class="font-semibold text-2xl">Inbox</span>
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarHeader>

          <SidebarSeparator />

          <SidebarContent>

            <!-- Group 1 -->
            <SidebarGroup>
              <!-- <SidebarGroupLabel>Conversations</SidebarGroupLabel> -->
              <SidebarMenu>

                <SidebarMenuItem>
                  <router-link :to="{ name: 'dashboard' }">
                    <SidebarMenuButton>
                      <FileLineChart />
                      <span>Dashboard</span>
                    </SidebarMenuButton>
                  </router-link>
                </SidebarMenuItem>

                <!-- Inbox -->
                <Collapsible defaultOpen class="group/collapsible" as-child>
                  <SidebarMenuItem>
                    <CollapsibleTrigger as-child>
                      <SidebarMenuButton tooltip="Inboxes">
                        <MessageCircle />
                        <span>Inboxes</span>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                      </SidebarMenuButton>
                    </CollapsibleTrigger>

                    <CollapsibleContent>
                      <SidebarMenuSub>
                        <SidebarMenuSubItem>
                          <SidebarMenuButton asChild>
                            <router-link
                              :to="{ name: 'conversations-list', params: { type: CONVERSATION_LIST_TYPE.ASSIGNED } }">
                              <span>My inbox</span>
                            </router-link>
                          </SidebarMenuButton>
                        </SidebarMenuSubItem>
                        <SidebarMenuSubItem>
                          <SidebarMenuButton asChild>
                            <router-link
                              :to="{ name: 'conversations-list', params: { type: CONVERSATION_LIST_TYPE.UNASSIGNED } }">
                              <span>Unassigned</span>
                            </router-link>
                          </SidebarMenuButton>
                        </SidebarMenuSubItem>
                        <SidebarMenuSubItem>
                          <SidebarMenuButton asChild>
                            <router-link
                              :to="{ name: 'conversations-list', params: { type: CONVERSATION_LIST_TYPE.ALL } }">
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
                    <CollapsibleTrigger asChild>
                      <div v-if="isLoading">
                        <SidebarMenuSkeleton showIcon v-for="i in 5" :key="i" />
                      </div>
                      <SidebarMenuButton asChild>
                        <a href="#">
                          <Users />
                          <span>Team inboxes</span>
                          <ChevronRight
                            class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                        </a>
                      </SidebarMenuButton>
                      <SidebarMenuAction>
                      </SidebarMenuAction>

                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        <SidebarMenuSubItem>
                          <SidebarMenuButton asChild v-for="team in userTeams" :key="team.id">
                            <router-link :to="{ name: 'conversations-team-list', params: { teamID: team.id } }">
                              <span>{{ team.name }}</span>
                            </router-link>
                          </SidebarMenuButton>
                        </SidebarMenuSubItem>
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </SidebarMenuItem>
                </Collapsible>

                <!-- Views -->
                <Collapsible defaultOpen class="group/collapsible">
                  <SidebarMenuItem>
                    <CollapsibleTrigger asChild>
                      <div v-if="isLoading">
                        <SidebarMenuSkeleton showIcon v-for="i in 5" :key="i" />
                      </div>
                      <SidebarMenuButton asChild>
                        <a href="#">
                          <SlidersHorizontal />
                          <span>Views</span>
                          <div>
                            <Plus size="18" @click="openCreateViewDialog" class="rounded-lg cursor-pointer opacity-0 transition-all duration-200 
              group-hover:opacity-100 hover:bg-gray-200 hover:shadow-sm
              text-gray-600 hover:text-gray-800 transform hover:scale-105 
              active:scale-100 p-1" />
                          </div>
                        </a>
                      </SidebarMenuButton>

                      <SidebarMenuAction>
                        <ChevronRight
                          class="transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                          v-if="userViews.length" />
                      </SidebarMenuAction>

                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub v-for="view in userViews" :key="view.id">
                        <SidebarMenuSubItem>
                          <SidebarMenuButton asChild>
                            <router-link :to="{ name: 'conversations-view-list', params: { viewID: view.id } }">
                              <span>{{ view.name }}</span>
                            </router-link>
                          </SidebarMenuButton>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <SidebarMenuAction>
                                <EllipsisVertical />
                              </SidebarMenuAction>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent side="right" align="start">
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

          <SidebarRail />

        </Sidebar>
      </template>

    </Sidebar>

    <!-- Main Content Area -->
    <SidebarInset>
      <slot></slot>
    </SidebarInset>
  </SidebarProvider>
</template>
