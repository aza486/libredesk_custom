<template>
  <div class="flex items-center space-x-4 px-2 h-12">
    <SidebarTrigger class="cursor-pointer w-4 h-4 text-black" />
    <div class="flex-1 flex items-center">
      <Search class="w-5 h-5" />
      <Separator orientation="vertical" />
      <Input
        v-model="model"
        placeholder="Search"
        class="w-full border-none shadow-none focus:ring-0 focus:ring-offset-0"
      />
    </div>
  </div>
  <Separator />
</template>

<script setup>
import { Separator } from '@/components/ui/separator'
import { Input } from '@/components/ui/input'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Search } from 'lucide-vue-next'
const model = defineModel(() => '')
</script>

<style scoped>
.focus\:ring-0:focus {
  --tw-ring-offset-shadow: none;
  --tw-ring-shadow: none;
  box-shadow: none;
}
</style>
