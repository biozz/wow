<script setup lang="ts">
// Welcome page for Durak card game
</script>

<template>
  <div class="text-center">
    <h1 class="text-4xl font-bold text-gray-900 mb-6">Welcome to Durak</h1>
    <p class="text-xl text-gray-600 mb-8">
      A traditional Russian card game where the goal is to get rid of all your cards.
    </p>
    
    <div class="space-y-4">
      <UButton 
        to="/lobbies" 
        color="primary" 
        size="lg"
        icon="i-lucide-users"
      >
        Join a Lobby
      </UButton>
      
      <div class="text-sm text-gray-500">
        <p>Create or join a lobby to start playing with friends!</p>
      </div>
    </div>
  </div>
</template>