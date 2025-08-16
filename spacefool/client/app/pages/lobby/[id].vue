<script setup lang="ts">
import { shallowRef, computed, watch, watchEffect, onMounted, onUnmounted, reactive } from 'vue';
import { DbConnection, Lobby, User, GameSettings } from '../../../module_bindings';
import { Identity } from '@clockworklabs/spacetimedb-sdk';
import { useRoute, useRouter } from 'vue-router';

// Route params
const route = useRoute();
const router = useRouter();
const lobbyId = computed(() => BigInt(route.params.id as string));

// Connection state
const connected = shallowRef<boolean>(false);
const identity = shallowRef<Identity | null>(null);
const conn = shallowRef<DbConnection | null>(null);
// CPU bot connection (hidden)
const botConn = shallowRef<DbConnection | null>(null);
const botIdentityHex = shallowRef<string | null>(null);

// Data state
const lobbies = shallowRef<Lobby[]>([]);
const users = shallowRef<Map<string, User>>(new Map());
const currentUser = shallowRef<User | null>(null);
const gameSettings = shallowRef<GameSettings | null>(null);

// UI state
const showSettings = shallowRef(false);
const settingName = shallowRef(false);
const isCpuMode = computed(() => typeof localStorage !== 'undefined' && localStorage.getItem('cpu_mode') === '1');

// Form states
const nameForm = reactive({
  name: ''
});

// Computed values
const currentLobby = computed(() => 
  lobbies.value.find(lobby => lobby.id === lobbyId.value)
);

const lobbyPlayers = computed(() => 
  Array.from(users.value.values()).filter(user => 
    user.currentLobbyId === lobbyId.value
  )
);

const isCreator = computed(() => 
  currentUser.value?.identity.toHexString() === currentLobby.value?.creator.toHexString()
);

const name = computed(() => {
  if (!identity.value) return 'unknown';
  return currentUser.value?.name || 
         identity.value.toHexString().substring(0, 8) || 
         'unknown';
});

const formatDate = (timestamp: any) => {
  // Convert timestamp to Date object
  const date = new Date(timestamp);
  return date.toLocaleDateString();
};

// Event handlers
let lobbyInsertHandler: ((ctx: any, lobby: Lobby) => void) | null = null;
let lobbyUpdateHandler: ((ctx: any, oldLobby: Lobby, newLobby: Lobby) => void) | null = null;
let lobbyDeleteHandler: ((ctx: any, lobby: Lobby) => void) | null = null;
let userInsertHandler: ((ctx: any, user: User) => void) | null = null;
let userUpdateHandler: ((ctx: any, oldUser: User, newUser: User) => void) | null = null;
let userDeleteHandler: ((ctx: any, user: User) => void) | null = null;
let gameSettingsUpdateHandler: ((ctx: any, oldSettings: GameSettings, newSettings: GameSettings) => void) | null = null;

onMounted(() => {
  const onConnect = (
    connInstance: DbConnection,
    identityInstance: Identity,
    token: string
  ) => {
    identity.value = identityInstance;
    connected.value = true;
    localStorage.setItem('auth_token', token);
    console.log('Connected to SpacetimeDB with identity:', identityInstance.toHexString());
    
    // Set up database event listeners
    lobbyInsertHandler = (_ctx: any, lobby: Lobby) => {
      lobbies.value = [...lobbies.value, lobby];
    };
    
    lobbyUpdateHandler = (_ctx: any, oldLobby: Lobby, newLobby: Lobby) => {
      lobbies.value = lobbies.value.map(l => l.id === newLobby.id ? newLobby : l);
    };

    lobbyDeleteHandler = (_ctx: any, lobby: Lobby) => {
      lobbies.value = lobbies.value.filter(l => l.id !== lobby.id);
    };

    userInsertHandler = (_ctx: any, user: User) => {
      users.value = new Map(users.value.set(user.identity.toHexString(), user));
      if (user.identity.toHexString() === identityInstance.toHexString()) {
        currentUser.value = user;
      }
    };

    userUpdateHandler = (_ctx: any, oldUser: User, newUser: User) => {
      users.value = new Map(users.value.set(newUser.identity.toHexString(), newUser));
      if (newUser.identity.toHexString() === identityInstance.toHexString()) {
        currentUser.value = newUser;
      }
    };

    userDeleteHandler = (_ctx: any, user: User) => {
      const newMap = new Map(users.value);
      newMap.delete(user.identity.toHexString());
      users.value = newMap;
    };

    gameSettingsUpdateHandler = (_ctx: any, oldSettings: GameSettings, newSettings: GameSettings) => {
      if (newSettings.lobbyId === lobbyId.value) {
        gameSettings.value = newSettings;
      }
    };

    // Register all the event handlers
    connInstance.db.lobby.onInsert(lobbyInsertHandler);
    connInstance.db.lobby.onUpdate(lobbyUpdateHandler);
    connInstance.db.lobby.onDelete(lobbyDeleteHandler);
    connInstance.db.user.onInsert(userInsertHandler);
    connInstance.db.user.onUpdate(userUpdateHandler);
    connInstance.db.user.onDelete(userDeleteHandler);
    connInstance.db.gameSettings.onUpdate(gameSettingsUpdateHandler);

    // Subscribe to queries
    connInstance
      ?.subscriptionBuilder()
      .onApplied(() => {
        console.log('SDK client cache initialized.');
      })
      .subscribe(['SELECT * FROM lobby', 'SELECT * FROM user', 'SELECT * FROM game_settings']);
  };

  const onDisconnect = () => {
    console.log('Disconnected from SpacetimeDB');
    connected.value = false;
  };

  const onConnectError = (_ctx: any, err: Error) => {
    console.log('Error connecting to SpacetimeDB:', err);
  };

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

  // If quick play requested, spawn CPU connection and orchestrate
  if (localStorage.getItem('cpu_mode') === '1') {
    const onBotConnect = (connInstance: DbConnection, botId: Identity, token: string) => {
      botIdentityHex.value = botId.toHexString();
      try { localStorage.setItem('cpu_bot_id', botIdentityHex.value); } catch (_) {}
      try { localStorage.setItem('cpu_bot_token', token); } catch (_) {}
      // Give the bot a name (best-effort)
      connInstance.reducers.setName('CPU');
      // Try to join this lobby
      // Delay a tick to ensure route param computed
      setTimeout(() => {
        try { connInstance.reducers.joinLobby(lobbyId.value); } catch (_) {}
      }, 50);
    };
    const onBotDisconnect = () => {};
    const onBotError = (_ctx: any, _err: Error) => {};
    botConn.value = (
      DbConnection.builder()
        .withUri('ws://localhost:3000')
        .withModuleName('spacefool')
        // Intentionally omit token + do NOT touch localStorage
        .onConnect(onBotConnect)
        .onDisconnect(onBotDisconnect)
        .onConnectError(onBotError)
        .build()
    );
  }
});

// Navigate to game when current user gets a gameId
watch(
  () => currentUser.value?.currentGameId,
  (newGameId) => {
    if (newGameId) {
      router.push(`/game/${newGameId.toString()}`);
    }
  }
);

onUnmounted(() => {
  // Clean up all event handlers
  if (conn.value) {
    if (lobbyInsertHandler) {
      conn.value.db.lobby.removeOnInsert(lobbyInsertHandler);
    }
    if (lobbyUpdateHandler) {
      conn.value.db.lobby.removeOnUpdate(lobbyUpdateHandler);
    }
    if (lobbyDeleteHandler) {
      conn.value.db.lobby.removeOnDelete(lobbyDeleteHandler);
    }
    if (userInsertHandler) {
      conn.value.db.user.removeOnInsert(userInsertHandler);
    }
    if (userUpdateHandler) {
      conn.value.db.user.removeOnUpdate(userUpdateHandler);
    }
    if (userDeleteHandler) {
      conn.value.db.user.removeOnDelete(userDeleteHandler);
    }
    if (gameSettingsUpdateHandler) {
      conn.value.db.gameSettings.removeOnUpdate(gameSettingsUpdateHandler);
    }
  }
});

// Actions
const leaveLobby = () => {
  conn.value?.reducers.leaveLobby();
  router.push('/lobbies');
};

const startGame = () => {
  if (currentLobby.value) {
    conn.value?.reducers.startGame(currentLobby.value.id);
  }
};

// Auto-start when CPU has joined and we are creator
watch(
  () => ({ count: lobbyPlayers.value.length, status: currentLobby.value?.status.tag, isCreator: isCreator.value, cpu: isCpuMode.value }),
  (s) => {
    if (s.cpu && s.isCreator && s.status === 'Waiting' && s.count >= 2) {
      startGame();
    }
  },
  { deep: true }
);

const updateGameSettings = (settings: Partial<GameSettings>) => {
  if (gameSettings.value) {
    conn.value?.reducers.updateGameSettings(
      lobbyId.value,
      settings.deckSize || gameSettings.value.deckSize,
      settings.startingCards || gameSettings.value.startingCards,
      settings.maxAttackCards || gameSettings.value.maxAttackCards,
      settings.multiRoundMode ?? gameSettings.value.multiRoundMode,
      settings.maxPoints || gameSettings.value.maxPoints,
      settings.anyoneCanAttack ?? gameSettings.value.anyoneCanAttack,
      settings.trumpCardToPlayer ?? gameSettings.value.trumpCardToPlayer
    );
  }
};

const onSubmitNewName = () => {
  settingName.value = false;
  conn.value?.reducers.setName(nameForm.name);
  nameForm.name = '';
};
</script>

<template>
  <div v-if="connected && currentLobby" class="min-h-screen p-4">
    <div class="max-w-6xl mx-auto">
      <!-- Header -->
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-3xl font-bold">{{ currentLobby.name }}</h1>
          <p class="text-gray-600">Lobby ID: {{ currentLobby.id.toString() }}</p>
        </div>
        
        <div class="flex items-center gap-4">
          <!-- User name -->
          <div v-if="!settingName" class="flex items-center gap-2">
            <span class="text-sm text-gray-600">{{ name }}</span>
            <UButton 
              @click="settingName = true; nameForm.name = name" 
              variant="outline"
              size="sm"
            >
              Edit Name
            </UButton>
          </div>
          
          <UForm v-else :state="nameForm" @submit="onSubmitNewName" class="flex gap-2">
            <UInput v-model="nameForm.name" size="sm" />
            <UButton type="submit" size="sm">Save</UButton>
          </UForm>

          <!-- Leave lobby button -->
          <UButton
            @click="leaveLobby"
            color="error"
            variant="outline"
            size="sm"
          >
            Leave Lobby
          </UButton>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Players Section -->
        <div class="lg:col-span-2">
          <UCard>
            <template #header>
              <div class="flex justify-between items-center">
                <h2 class="text-xl font-semibold">Players</h2>
                <UBadge 
                  :color="currentLobby.status.tag === 'Waiting' ? 'success' : 'warning'"
                  variant="subtle"
                >
                  {{ currentLobby.status.tag }}
                </UBadge>
              </div>
            </template>

            <div class="space-y-4">
              <div class="flex justify-between text-sm text-gray-600">
                <span>{{ lobbyPlayers.length }}/{{ currentLobby.maxPlayers }} players</span>
                <span>Created: {{ formatDate(currentLobby.createdAt) }}</span>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div 
                  v-for="player in lobbyPlayers" 
                  :key="player.identity.toHexString()"
                  class="flex items-center justify-between p-3 border rounded-lg"
                  :class="player.identity.toHexString() === currentLobby.creator.toHexString() ? 'border-primary bg-primary/5' : 'border-gray-200'"
                >
                  <div class="flex items-center gap-2">
                    <UIcon 
                      :name="player.online ? 'i-lucide-circle' : 'i-lucide-circle-off'" 
                      :class="player.online ? 'text-success' : 'text-gray-400'"
                    />
                    <span class="font-medium">
                      {{ player.name || player.identity.toHexString().substring(0, 8) }}
                    </span>
                    <UBadge 
                      v-if="player.identity.toHexString() === currentLobby.creator.toHexString()"
                      color="primary"
                      variant="subtle"
                      size="sm"
                    >
                      Creator
                    </UBadge>
                    <UBadge 
                      v-if="botIdentityHex && player.identity.toHexString() === botIdentityHex"
                      color="neutral"
                      variant="subtle"
                      size="sm"
                    >
                      CPU
                    </UBadge>
                  </div>
                </div>
              </div>

              <!-- Start Game Button / Auto-start in CPU mode -->
              <div v-if="isCreator && currentLobby.status.tag === 'Waiting' && lobbyPlayers.length >= 2" class="flex justify-center pt-4">
                <UButton
                  @click="startGame"
                  color="primary"
                  size="lg"
                  icon="i-lucide-play"
                >
                  Start Game
                </UButton>
              </div>
              <div v-else-if="isCreator && currentLobby.status.tag === 'Waiting' && lobbyPlayers.length >= 2 && isCpuMode" class="flex justify-center pt-4">
                <UButton
                  color="primary"
                  size="lg"
                  icon="i-lucide-play"
                  @click="startGame"
                >
                  Waiting for auto-start...
                </UButton>
              </div>
            </div>
          </UCard>
        </div>

        <!-- Settings Section -->
        <div class="space-y-6">
          <!-- Game Settings -->
          <UCard>
            <template #header>
              <div class="flex justify-between items-center">
                <h3 class="text-lg font-semibold">Game Settings</h3>
                <UButton
                  v-if="isCreator"
                  @click="showSettings = !showSettings"
                  :icon="showSettings ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                  variant="outline"
                  size="sm"
                >
                  {{ showSettings ? 'Hide' : 'Edit' }}
                </UButton>
              </div>
            </template>

            <div v-if="gameSettings" class="space-y-3">
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="font-medium">Deck Size:</span>
                  <span class="ml-2">{{ gameSettings.deckSize.tag === 'Standard36' ? '36 cards' : '52 cards' }}</span>
                </div>
                <div>
                  <span class="font-medium">Starting Cards:</span>
                  <span class="ml-2">{{ gameSettings.startingCards }}</span>
                </div>
                <div>
                  <span class="font-medium">Max Attack Cards:</span>
                  <span class="ml-2">{{ gameSettings.maxAttackCards === 0 ? 'No limit' : gameSettings.maxAttackCards }}</span>
                </div>
                <div>
                  <span class="font-medium">Multi Round:</span>
                  <span class="ml-2">{{ gameSettings.multiRoundMode ? 'Yes' : 'No' }}</span>
                </div>
                <div v-if="gameSettings.multiRoundMode">
                  <span class="font-medium">Max Points:</span>
                  <span class="ml-2">{{ gameSettings.maxPoints }}</span>
                </div>
                <div>
                  <span class="font-medium">Anyone Can Attack:</span>
                  <span class="ml-2">{{ gameSettings.anyoneCanAttack ? 'Yes' : 'No' }}</span>
                </div>
              </div>

              <div v-if="showSettings && isCreator" class="pt-4 border-t">
                <GameSettings
                  :settings="gameSettings"
                  :is-creator="isCreator"
                  :on-update="updateGameSettings"
                />
              </div>
            </div>

            <div v-else class="text-center py-4 text-gray-500">
              <p>No game settings available</p>
            </div>
          </UCard>

          <!-- Quick Actions -->
          <UCard>
            <template #header>
              <h3 class="text-lg font-semibold">Quick Actions</h3>
            </template>

            <div class="space-y-3">
              <UButton
                @click="router.push('/lobbies')"
                variant="outline"
                block
                icon="i-lucide-arrow-left"
              >
                Back to Lobbies
              </UButton>
            </div>
          </UCard>
        </div>
      </div>
    </div>
  </div>
  
  <div v-else-if="connected && !currentLobby" class="min-h-screen flex items-center justify-center">
    <div class="text-center">
      <p class="text-lg text-gray-600 mb-4">Lobby not found or you don't have access</p>
      <UButton @click="router.push('/lobbies')" color="primary">
        Back to Lobbies
      </UButton>
    </div>
  </div>
  
  <div v-else class="min-h-screen flex items-center justify-center">
    <p class="text-lg">Connecting to SpacetimeDB...</p>
  </div>
</template> 