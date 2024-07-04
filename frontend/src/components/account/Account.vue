<script setup>
import { computed } from 'vue'
import SidebarNav from '@/components/account/SidebarNav.vue'
import { Separator } from '@/components/ui/separator'

const props = defineProps({
    page: {
        type: String,
        required: true
    }
})

const currentPage = computed(() => {
    return props.page
})
</script>

<template>
    <div class="space-y-6 md:block page-content">
        <div class="space-y-0.5">
            <h2 class="text-2xl font-bold tracking-tight">
                Profile settings
            </h2>
            <p class="text-muted-foreground">
                Manage your account settings and set e-mail preferences.
            </p>
        </div>
        <Separator class="my-6" />
        <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
            <aside class="-mx-4 lg:w-1/5">
                <SidebarNav :current-page="currentPage" />
            </aside>
            <div class="flex-1 lg:max-w-2xl">
                <div class="space-y-6">
                    <slot :page="currentPage" />
                </div>
            </div>
        </div>
    </div>
</template>
