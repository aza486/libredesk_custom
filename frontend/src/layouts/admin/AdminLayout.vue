<template>
  <main class="p-6 page-content pb-12">
    <div class="max-w-7xl mx-auto">
      <div class="flex gap-6 justify-start">
        <div class="flex-grow">
          <div class="bg-white shadow-md rounded-lg overflow-hidden">
            <div class="p-6 sm:p-8">
              <router-view />
            </div>
          </div>
        </div>
      </div>
    </div>
  </main>
</template>
