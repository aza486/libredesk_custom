<script setup>
import Admin from '@/components/admin/AdminPage.vue'
</script>

<template>
  <Admin class="page-content">
    <main class="p-6 lg:p-8">
      <div class="max-w-6xl mx-auto">
        <div class="bg-white shadow-md rounded-lg overflow-hidden">
          <div class="p-6 sm:p-8">
            <div class="space-y-6">
              <router-view></router-view>
            </div>
          </div>
        </div>
      </div>
    </main>
  </Admin>
</template>
