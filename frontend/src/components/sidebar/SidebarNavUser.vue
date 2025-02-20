<template>
    <DropdownMenu>
        <DropdownMenuTrigger as-child>
            <SidebarMenuButton size="lg"
                class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground p-0">
                <Avatar class="h-8 w-8 rounded-lg">
                    <AvatarImage :src="userStore.avatar" alt="Abhinav" />
                    <AvatarFallback class="rounded-lg">
                        {{ userStore.getInitials }}
                    </AvatarFallback>
                </Avatar>
                <div class="grid flex-1 text-left text-sm leading-tight">
                    <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
                    <span class="truncate text-xs">{{ userStore.email }}</span>
                </div>
                <ChevronsUpDown class="ml-auto size-4" />
            </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent class="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg" side="bottom"
            :side-offset="4">
            <DropdownMenuLabel class="p-0 font-normal">
                <div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <Avatar class="h-8 w-8 rounded-lg">
                        <AvatarImage :src="userStore.avatar" alt="Abhinav" />
                        <AvatarFallback class="rounded-lg">
                            {{ userStore.getInitials }}
                        </AvatarFallback>
                    </Avatar>
                    <div class="grid flex-1 text-left text-sm leading-tight">
                        <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
                        <span class="truncate text-xs">{{ userStore.email }}</span>
                    </div>
                </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
                <DropdownMenuItem>
                    <router-link to="/account" class="flex items-center">
                        <CircleUserRound size="18" class="mr-2" />
                        Account
                    </router-link>
                </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem @click="logout">
                <LogOut size="18" class="mr-2" />
                Log out
            </DropdownMenuItem>
        </DropdownMenuContent>
    </DropdownMenu>
</template>

<script setup>
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
    SidebarMenuButton,
} from '@/components/ui/sidebar'
import {
    Avatar,
    AvatarFallback,
    AvatarImage,
} from '@/components/ui/avatar'
import {
    ChevronsUpDown,
    CircleUserRound,
    LogOut,
} from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()

const logout = () => {
    window.location.href = '/logout'
}
</script>