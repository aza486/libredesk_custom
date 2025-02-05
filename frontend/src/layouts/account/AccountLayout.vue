<template>
  <div class="space-y-4 md:block page-content">
    <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
      <div class="flex-1 lg:max-w-3xl admin-main-content min-h-[700px]">
        <div class="space-y-6">
          <router-view />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup></script>
