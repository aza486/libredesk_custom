<script setup>
import { MoreHorizontal } from 'lucide-vue-next'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'

const router = useRouter()

const props = defineProps({
    tag: {
        type: Object,
        required: true,
        default: () => ({
            id: ''
        })
    }
})


</script>

<template>
    <DropdownMenu>
        <DropdownMenuTrigger as-child>
            <Button variant="ghost" class="w-8 h-8 p-0">
                <span class="sr-only">Open menu</span>
                <MoreHorizontal class="w-4 h-4" />
            </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
            <DropdownMenuItem>Edit</DropdownMenuItem>
            <DropdownMenuItem>Delete</DropdownMenuItem>
        </DropdownMenuContent>
    </DropdownMenu>
</template>
