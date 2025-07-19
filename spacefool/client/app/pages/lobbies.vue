<script setup lang="ts">
import { shallowRef, computed, watch, watchEffect, onMounted, onUnmounted, reactive } from 'vue';
import { DbConnection, Lobby, User } from '../../module_bindings';
import { Identity } from '@clockworklabs/spacetimedb-sdk';

// Connection state
const connected = shallowRef<boolean>(false);
const identity = shallowRef<Identity | null>(null);
const conn = shallowRef<DbConnection | null>(null);

// Data state
const lobbies = shallowRef<Lobby[]>([]);
const users = shallowRef<Map<string, User>>(new Map());
const currentUser = shallowRef<User | null>(null);

// Form states
const createLobbyForm = reactive({
  name: '',
  maxPlayers: 4
});

const nameForm = reactive({
  name: ''
});

// UI state
const showCreateLobby = shallowRef(false);
const settingName = shallowRef(false);

// Computed values
const availableLobbies = computed(() => 
  lobbies.value.filter(lobby => lobby.status.tag === 'Waiting')
);

const lobbyPlayers = computed(() => (lobbyId: bigint) => {
  return Array.from(users.value.values()).filter(user => 
    user.currentLobbyId === lobbyId
  );
});

const name = computed(() => {
  if (!identity.value) return 'unknown';
  return currentUser.value?.name || 
         identity.value.toHexString().substring(0, 8) || 
         'unknown';
});

// Event handlers
let lobbyInsertHandler: ((ctx: any, lobby: Lobby) => void) | null = null;
let lobbyUpdateHandler: ((ctx: any, oldLobby: Lobby, newLobby: Lobby) => void) | null = null;
let lobbyDeleteHandler: ((ctx: any, lobby: Lobby) => void) | null = null;
let userInsertHandler: ((ctx: any, user: User) => void) | null = null;
let userUpdateHandler: ((ctx: any, oldUser: User, newUser: User) => void) | null = null;
let userDeleteHandler: ((ctx: any, user: User) => void) | null = null;

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

    // Register all the event handlers
    connInstance.db.lobby.onInsert(lobbyInsertHandler);
    connInstance.db.lobby.onUpdate(lobbyUpdateHandler);
    connInstance.db.lobby.onDelete(lobbyDeleteHandler);
    connInstance.db.user.onInsert(userInsertHandler);
    connInstance.db.user.onUpdate(userUpdateHandler);
    connInstance.db.user.onDelete(userDeleteHandler);

    // Subscribe to queries
    connInstance
      ?.subscriptionBuilder()
      .onApplied(() => {
        console.log('SDK client cache initialized.');
      })
      .subscribe(['SELECT * FROM lobby', 'SELECT * FROM user']);
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
});

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
  }
});

// Actions
const createLobby = () => {
  if (createLobbyForm.name.trim() && createLobbyForm.maxPlayers >= 2 && createLobbyForm.maxPlayers <= 6) {
    conn.value?.reducers.createLobby(createLobbyForm.name, createLobbyForm.maxPlayers);
    createLobbyForm.name = '';
    createLobbyForm.maxPlayers = 4;
    showCreateLobby.value = false;
  }
};

const joinLobby = (lobbyId: bigint) => {
  conn.value?.reducers.joinLobby(lobbyId);
};

const leaveLobby = () => {
  conn.value?.reducers.leaveLobby();
};

const startGame = (lobbyId: bigint) => {
  conn.value?.reducers.startGame(lobbyId);
};

const deleteLobby = (lobbyId: bigint) => {
  // This would need to be implemented in the backend
  console.log('Delete lobby:', lobbyId);
};

const onSubmitNewName = () => {
  settingName.value = false;
  conn.value?.reducers.setName(nameForm.name);
  nameForm.name = '';
};
</script>

<template>
  <div v-if="connected" class="min-h-screen p-4">
    <div class="max-w-6xl mx-auto">
      <!-- Header -->
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold">Durak Lobbies</h1>
        
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
            v-if="currentUser?.currentLobbyId"
            @click="leaveLobby"
            color="error"
            variant="outline"
            size="sm"
          >
            Leave Lobby
          </UButton>
        </div>
      </div>

      <!-- Create Lobby Section -->
      <UCard class="mb-6">
        <template #header>
          <div class="flex justify-between items-center">
            <h2 class="text-xl font-semibold">Create New Lobby</h2>
            <UButton
              @click="showCreateLobby = !showCreateLobby"
              :icon="showCreateLobby ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
              variant="outline"
              size="sm"
            >
              {{ showCreateLobby ? 'Hide' : 'Show' }}
            </UButton>
          </div>
        </template>

        <div v-if="showCreateLobby" class="space-y-4">
          <UForm :state="createLobbyForm" @submit="createLobby">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <UFormGroup label="Lobby Name" name="name">
                <UInput 
                  v-model="createLobbyForm.name" 
                  placeholder="Enter lobby name..."
                />
              </UFormGroup>
              
              <UFormGroup label="Max Players" name="maxPlayers">
                <UInputNumber
                  v-model="createLobbyForm.maxPlayers"
                  :min="2"
                  :max="6"
                />
              </UFormGroup>
            </div>
            
            <div class="flex justify-end">
              <UButton type="submit" color="primary">
                Create Lobby
              </UButton>
            </div>
          </UForm>
        </div>
      </UCard>

      <!-- Available Lobbies -->
      <div class="space-y-4">
        <h2 class="text-xl font-semibold">Available Lobbies</h2>
        
        <div v-if="availableLobbies.length === 0" class="text-center py-8">
          <p class="text-gray-500">No available lobbies. Create one to get started!</p>
        </div>
        
        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <LobbyCard
            v-for="lobby in availableLobbies"
            :key="lobby.id.toString()"
            :lobby="lobby"
            :players="lobbyPlayers(lobby.id)"
            :current-user="currentUser"
            :on-join="joinLobby"
            @start-game="startGame"
            @delete-lobby="deleteLobby"
          />
        </div>
      </div>
    </div>
  </div>
  
  <div v-else class="min-h-screen flex items-center justify-center">
    <p class="text-lg">Connecting to SpacetimeDB...</p>
  </div>
</template> 