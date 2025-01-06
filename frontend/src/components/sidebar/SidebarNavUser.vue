<template>
    <DropdownMenu>
        <DropdownMenuTrigger as-child>
            <SidebarMenuButton size="lg"
                class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground p-0">
                <Avatar class="h-8 w-8 rounded-lg">
                    <AvatarImage src="https://avatars.githubusercontent.com/u/48166553?v=4" alt="Abhinav" />
                    <AvatarFallback class="rounded-lg">
                        AR
                    </AvatarFallback>
                </Avatar>
                <div class="grid flex-1 text-left text-sm leading-tight">
                    <span class="truncate font-semibold">Abhinav Raut</span>
                    <span class="truncate text-xs">abhinavrautcs@gmail.com</span>
                </div>
                <ChevronsUpDown class="ml-auto size-4" />
            </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent class="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg" side="bottom"
            align="end" :side-offset="4">
            <DropdownMenuLabel class="p-0 font-normal">
                <div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <Avatar class="h-8 w-8 rounded-lg">
                        <AvatarImage src="https://avatars.githubusercontent.com/u/48166553?v=4" alt="Abhinav" />
                        <AvatarFallback class="rounded-lg">
                            AR
                        </AvatarFallback>
                    </Avatar>
                    <div class="grid flex-1 text-left text-sm leading-tight">
                        <span class="truncate font-semibold">Abhinav Raut!</span>
                        <span class="truncate text-xs">abhinavrautcs@gmail.com</span>
                    </div>
                </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <!-- <DropdownMenuGroup>
                <DropdownMenuItem>
                    <Sparkles size="18" class="mr-2" />
                    Upgrade to Pro
                </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator /> -->
            <DropdownMenuGroup>
                <DropdownMenuItem>
                    <router-link to="/account" class="flex items-center">
                        <CircleUserRound size="18" class="mr-2" />
                        Account
                    </router-link>
                </DropdownMenuItem>
                <!-- <DropdownMenuItem>
                    <Bell size="18" class="mr-2" />
                    Notifications
                </DropdownMenuItem> -->
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem @click.stop="window.location.href = '/logout'">
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
    AudioWaveform,
    BadgeCheck,
    Bell,
    Users,
    CircleUserRound,
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
    Forward,
    EllipsisVertical,
    MessageCircleHeart,
    Plus,
    BarChart,
    MessageCircle,
    Frame,
    GalleryVerticalEnd,
    LogOut,
    Map,
    MoreHorizontal,
    PieChart,
    Settings2,
    Sparkles,
    ListFilter,
    SquareTerminal,
    Trash2,
} from 'lucide-vue-next'
</script>