<script setup lang="ts">
// Simple navigation component
</script>

<template>
  <nav class="bg-white shadow-sm border-b">
    <div class="max-w-6xl mx-auto px-4">
      <div class="flex justify-between items-center h-16">
        <div class="flex items-center">
          <NuxtLink to="/" class="text-xl font-bold text-gray-900">
            Durak
          </NuxtLink>
        </div>
        
        <div class="flex items-center space-x-4">
          <NuxtLink 
            to="/lobbies" 
            class="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
          >
            Lobbies
          </NuxtLink>
          <NuxtLink 
            to="/lobbies" 
            class="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
          >
            Play
          </NuxtLink>
        </div>
      </div>
    </div>
  </nav>
</template> 