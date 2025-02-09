<template>
    <Toaster />
    <TooltipProvider :delay-duration="150">
        <div class="!font-jakarta">
            <RouterView />
        </div>
    </TooltipProvider>
</template>

<script setup>
import { RouterView } from 'vue-router'
import { Toaster } from '@/components/ui/toast'
import { TooltipProvider } from '@/components/ui/tooltip'
</script>