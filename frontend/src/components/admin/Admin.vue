<script setup>
import SidebarNav from '@/components/admin/SidebarNav.vue'
import { Separator } from '@/components/ui/separator'
</script>

<template>
  <div class="space-y-4 md:block page-content">
    <div class="space-y-0.5">
      <h4>Admin settings</h4>
      <p class="text-muted-foreground text-sm">Manage your support desk settings.</p>
    </div>
    <Separator class="my-3" />
    <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
      <aside class="lg:w-1/6 md:w-1/7 h-full">
        <SidebarNav />
      </aside>
      <div class="flex-1 lg:max-w-4xl p-7 min-h-[700px]">
        <div class="space-y-6">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>
