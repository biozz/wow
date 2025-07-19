<script setup lang="ts">
import { computed } from 'vue';
import { DbConnection, Lobby, User, LobbyStatus } from '../../module_bindings';

interface Props {
  lobby: Lobby;
  players: User[];
  currentUser: User | null;
  onJoin: (lobbyId: bigint) => void;
}

const props = defineProps<Props>();

const isCreator = computed(() => 
  props.currentUser?.identity.toHexString() === props.lobby.creator.toHexString()
);

const canJoin = computed(() => 
  props.currentUser && 
  !props.currentUser.currentLobbyId && 
  !props.currentUser.currentGameId &&
  props.lobby.status.tag === 'Waiting' &&
  props.lobby.currentPlayers < props.lobby.maxPlayers
);

const isWaiting = computed(() => props.lobby.status.tag === 'Waiting');

const formatDate = (timestamp: any) => {
  // Convert timestamp to Date object
  const date = new Date(timestamp);
  return date.toLocaleDateString();
};
</script>

<template>
  <UCard class="w-full">
    <template #header>
      <div class="flex justify-between items-center">
        <h3 class="text-lg font-semibold">{{ lobby.name }}</h3>
        <UBadge 
          :color="isWaiting ? 'success' : 'warning'"
          variant="subtle"
        >
          {{ lobby.status.tag }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-3">
      <div class="flex justify-between text-sm text-gray-600">
        <span>Players: {{ lobby.currentPlayers }}/{{ lobby.maxPlayers }}</span>
        <span>Created: {{ formatDate(lobby.createdAt) }}</span>
      </div>

      <div v-if="players.length > 0" class="space-y-2">
        <h4 class="font-medium text-sm">Players:</h4>
        <div class="flex flex-wrap gap-2">
          <UBadge 
            v-for="player in players" 
            :key="player.identity.toHexString()"
            :color="player.online ? 'success' : 'neutral'"
            variant="subtle"
            size="sm"
          >
            {{ player.name || player.identity.toHexString().substring(0, 8) }}
            <UIcon 
              :name="player.online ? 'i-lucide-circle' : 'i-lucide-circle-off'" 
              class="ml-1"
            />
          </UBadge>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-between items-center">
        <span class="text-sm text-gray-500">
          Created by {{ players.find(p => p.identity.toHexString() === lobby.creator.toHexString())?.name || lobby.creator.toHexString().substring(0, 8) }}
        </span>
        
        <div class="flex gap-2">
          <UButton
            v-if="isCreator && isWaiting"
            color="primary"
            size="sm"
            @click="$emit('startGame', lobby.id)"
          >
            Start Game
          </UButton>
          
          <UButton
            v-if="canJoin"
            color="primary"
            size="sm"
            @click="onJoin(lobby.id)"
          >
            Join
          </UButton>
          
          <UButton
            v-if="isCreator && isWaiting"
            color="error"
            variant="outline"
            size="sm"
            @click="$emit('deleteLobby', lobby.id)"
          >
            Delete
          </UButton>
        </div>
      </div>
    </template>
  </UCard>
</template> 