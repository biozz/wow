<script setup lang="ts">
import { shallowRef, onMounted } from 'vue';
import { DbConnection, type ReducerEventContext } from '../../module_bindings';
import type { Identity } from '@clockworklabs/spacetimedb-sdk';
import { useRouter } from 'vue-router';

const router = useRouter();
const connected = shallowRef(false);
const conn = shallowRef<DbConnection | null>(null);

const quickPlay = async () => {
  // CPU mode: create a 1v1 lobby and flag CPU orchestration
  try {
    localStorage.setItem('cpu_mode', '1');
    await conn.value?.reducers.createLobby('Solo vs Computer', 2);
  } finally {
    router.push('/lobbies');
  }
};

onMounted(() => {
  const onConnect = (connInstance: DbConnection, _identity: Identity, token: string) => {
    connected.value = true;
    localStorage.setItem('auth_token', token);
  };
  const onDisconnect = () => { connected.value = false; };
  const onConnectError = (_ctx: any, err: Error) => { console.error('Spacetime connect error', err); };
  conn.value = (
    DbConnection.builder()
      .withUri('ws://localhost:3000')
      .withModuleName('spacefool')
      .withToken(localStorage.getItem('auth_token') || '')
      .onConnect(onConnect)
      .onDisconnect(onDisconnect)
      .onConnectError(onConnectError)
      .build()
  );
});
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
      <UButton 
        :disabled="!connected"
        color="success" 
        size="lg"
        icon="i-lucide-rocket"
        @click="quickPlay"
      >
        Quick Play
      </UButton>
      
      <div class="text-sm text-gray-500">
        <p>Create or join a lobby to start playing with friends!</p>
        <p class="mt-1">Connected: {{ connected ? 'Yes' : 'No' }}</p>
      </div>
    </div>
  </div>
</template>