<template>
    <Toaster />
    <TooltipProvider :delay-duration="250">
        <div class="font-inter">
            <RouterView />
        </div>
    </TooltipProvider>
</template>

<script setup>
import { RouterView } from 'vue-router'
import { Toaster } from '@/components/ui/toast'
import { TooltipProvider } from '@/components/ui/tooltip'
</script>