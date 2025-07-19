<script setup lang="ts">
// Default layout with navigation
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <Navigation />
    <UContainer class="py-8">
      <slot />
    </UContainer>
  </div>
</template>