<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <SidebarMenuButton
        size="lg"
        class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground p-0"
      >
        <Avatar class="h-8 w-8 rounded-lg relative overflow-visible">
          <AvatarImage :src="userStore.avatar" alt="" class="rounded-lg" />
          <AvatarFallback class="rounded-lg">
            {{ userStore.getInitials }}
          </AvatarFallback>
          <div
            class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border border-background"
            :class="{
              'bg-green-500': userStore.user.availability_status === 'online',
              'bg-amber-500':
                userStore.user.availability_status === 'away' ||
                userStore.user.availability_status === 'away_manual' ||
                userStore.user.availability_status === 'away_and_reassigning',
              'bg-gray-400': userStore.user.availability_status === 'offline'
            }"
          ></div>
        </Avatar>
        <div class="grid flex-1 text-left text-sm leading-tight">
          <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
          <span class="truncate text-xs">{{ userStore.email }}</span>
        </div>
        <ChevronsUpDown class="ml-auto size-4" />
      </SidebarMenuButton>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      class="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
      side="bottom"
      :side-offset="4"
    >
      <DropdownMenuLabel class="p-0 font-normal space-y-1">
        <div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
          <Avatar class="h-8 w-8 rounded-lg">
            <AvatarImage :src="userStore.avatar" alt="U" />
            <AvatarFallback class="rounded-lg">
              {{ userStore.getInitials }}
            </AvatarFallback>
          </Avatar>
          <div class="grid flex-1 text-left text-sm leading-tight">
            <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
            <span class="truncate text-xs">{{ userStore.email }}</span>
          </div>
        </div>
        <div class="space-y-2">
          <!-- Away switch is checked with 'away_manual' or 'away_and_reassigning' -->
          <div class="flex items-center gap-2 px-1 text-left text-sm justify-between">
            <span class="text-muted-foreground">{{ t('navigation.away') }}</span>
            <Switch
              :checked="
                ['away_manual', 'away_and_reassigning'].includes(userStore.user.availability_status)
              "
              @update:checked="
                (val) => {
                  const newStatus = val ? 'away_manual' : 'online'
                  userStore.updateUserAvailability(newStatus)
                }
              "
            />
          </div>
          <!-- Reassign Replies Switch is checked with 'away_and_reassigning' -->
          <div class="flex items-center gap-2 px-1 text-left text-sm justify-between">
            <span class="text-muted-foreground">{{ t('navigation.reassign_replies') }}</span>
            <Switch
              :checked="userStore.user.availability_status === 'away_and_reassigning'"
              @update:checked="
                (val) => {
                  const newStatus = val ? 'away_and_reassigning' : 'away_manual'
                  userStore.updateUserAvailability(newStatus)
                }
              "
            />
          </div>
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem @click.prevent="router.push({ name: 'account' })">
          <CircleUserRound size="18" class="mr-2" />
          {{ t('navigation.account') }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="logout">
        <LogOut size="18" class="mr-2" />
        {{ t('navigation.logout') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { SidebarMenuButton } from '@/components/ui/sidebar'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Switch } from '@/components/ui/switch'
import { ChevronsUpDown, CircleUserRound, LogOut } from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'
import { useRouter } from 'vue-router'

const userStore = useUserStore()
const router = useRouter()
const { t } = useI18n()

const logout = () => {
  window.location.href = '/logout'
}
</script>
