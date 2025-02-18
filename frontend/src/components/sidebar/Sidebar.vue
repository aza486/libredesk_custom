<script setup>
import { adminNavItems, reportsNavItems, accountNavItems } from '@/constants/navigation'
import { RouterLink, useRoute } from 'vue-router'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarSeparator,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarRail
} from '@/components/ui/sidebar'
import {
  ChevronRight,
  EllipsisVertical,
  Plus,
  CircleUserRound,
  UserSearch,
  UsersRound,
  Search
} from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { filterNavItems } from '@/utils/nav-permissions'
import { useStorage } from '@vueuse/core'
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'

defineProps({
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] }
})
const userStore = useUserStore()
const route = useRoute()
const emit = defineEmits(['createView', 'editView', 'deleteView'])

const openCreateViewDialog = () => {
  emit('createView')
}

const editView = (view) => {
  emit('editView', view)
}

const deleteView = (view) => {
  emit('deleteView', view)
}

const filteredAdminNavItems = computed(() => filterNavItems(adminNavItems, userStore.can))
const filteredReportsNavItems = computed(() => filterNavItems(reportsNavItems, userStore.can))

const isActiveParent = (parentHref) => {
  return route.path.startsWith(parentHref)
}

const isInboxRoute = (path) => {
  return path.startsWith('/inboxes')
}

const sidebarOpen = useStorage('mainSidebarOpen', true)
</script>

<template>
  <SidebarProvider
    style="--sidebar-width: 14rem"
    :default-open="sidebarOpen"
    v-on:update:open="sidebarOpen = $event"
  >
    <!-- Reports sidebar -->
    <template
      v-if="
        userStore.hasReportTabPermissions &&
        route.matched.some((record) => record.name && record.name.startsWith('reports'))
      "
    >
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton :isActive="isActiveParent('/reports/overview')" asChild>
                <div>
                  <span class="font-semibold text-xl">Reports</span>
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
    <template v-if="route.matched.some((record) => record.name && record.name.startsWith('admin'))">
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton :isActive="isActiveParent('/admin')" asChild>
                <div>
                  <span class="font-semibold text-xl">Admin</span>
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
                        <SidebarMenuButton size="sm" :isActive="isActiveParent(child.href)" asChild>
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
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton :isActive="isActiveParent('/account/profile')" asChild>
                <div>
                  <span class="font-semibold text-xl">Account</span>
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

    <!-- Inbox sidebar -->
    <template v-if="route.path && isInboxRoute(route.path)">
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild>
                <div class="flex items-center justify-between w-full">
                  <div class="font-semibold text-xl">Inbox</div>
                  <div class="ml-auto">
                    <router-link :to="{ name: 'search' }">
                      <div class="flex items-center bg-accent p-2 rounded-full">
                        <Search
                          class="transition-transform duration-200 hover:scale-110 cursor-pointer"
                          size="15"
                          stroke-width="2.5"
                        />
                      </div>
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
              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/assigned')">
                  <router-link :to="{ name: 'inbox', params: { type: 'assigned' } }">
                    <CircleUserRound />
                    <span>My inbox</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/unassigned')">
                  <router-link :to="{ name: 'inbox', params: { type: 'unassigned' } }">
                    <UserSearch />
                    <span>Unassigned</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/all')">
                  <router-link :to="{ name: 'inbox', params: { type: 'all' } }">
                    <UsersRound />
                    <span>All</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Team Inboxes -->
              <Collapsible defaultOpen class="group/collapsible" v-if="userTeams.length">
                <SidebarMenuItem>
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton asChild>
                      <router-link to="#">
                        <!-- <Users /> -->
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
                        <SidebarMenuButton
                          size="sm"
                          :is-active="route.params.teamID == team.id"
                          asChild
                        >
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
                        <!-- <SlidersHorizontal /> -->
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
                        <SidebarMenuButton
                          size="sm"
                          :isActive="route.params.viewID == view.id"
                          asChild
                        >
                          <router-link :to="{ name: 'view-inbox', params: { viewID: view.id } }">
                            <span class="break-all w-24">{{ view.name }}</span>
                          </router-link>
                        </SidebarMenuButton>

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

    <!-- Main Content Area -->
    <SidebarInset>
      <slot></slot>
    </SidebarInset>
  </SidebarProvider>
</template>
