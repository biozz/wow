<script setup lang="ts">
import { DbConnection, type ErrorContext, type EventContext, Message, User } from '../../module_bindings';
import { Identity } from '@clockworklabs/spacetimedb-sdk';
import { shallowRef, computed, watch, watchEffect, onMounted, onUnmounted, reactive } from 'vue';

export type PrettyMessage = {
  senderName: string;
  text: string;
};

const settingName = shallowRef(false);
const systemMessage = shallowRef('');

// Form states for UForm components
const messageFormState = reactive({
  message: ''
});

const nameFormState = reactive({
  name: ''
});

const connected = shallowRef<boolean>(false);
const identity = shallowRef<Identity | null>(null);
const conn = shallowRef<DbConnection | null>(null);

// Local reactive state for messages and users
const messages = shallowRef<Message[]>([]);
const users = shallowRef<Map<string, User>>(new Map());

// Watch for user changes to update system messages
watchEffect(() => {
  if (!conn.value) return;

  const connection = conn.value;

  // Set up user listeners for system messages
  const onUserInsert = (_ctx: EventContext, user: User) => {
    if (user.online) {
      const name = user.name || user.identity.toHexString().substring(0, 8);
      systemMessage.value = systemMessage.value + `\n${name} has connected.`;
    }
  };

  const onUserUpdate = (_ctx: EventContext, oldUser: User, newUser: User) => {
    const name = newUser.name || newUser.identity.toHexString().substring(0, 8);
    if (oldUser.online === false && newUser.online === true) {
      systemMessage.value = systemMessage.value + `\n${name} has connected.`;
    } else if (oldUser.online === true && newUser.online === false) {
      systemMessage.value = systemMessage.value + `\n${name} has disconnected.`;
    }
  };

  connection.db.user.onInsert(onUserInsert);
  connection.db.user.onUpdate(onUserUpdate);

  // Cleanup function
  return () => {
    connection.db.user.removeOnInsert(onUserInsert);
    connection.db.user.removeOnUpdate(onUserUpdate);
  };
});

// Watch for connection establishment to set up subscriptions
watch(conn, (newConn) => {
  if (!newConn) return;
  
  newConn.subscriptionBuilder().onApplied(() => {
    console.log('SDK client cache initialized.');
  });
});

// Computed values
const prettyMessages = computed<PrettyMessage[]>(() => {
  return messages.value
    .sort((a: Message, b: Message) => (a.sent > b.sent ? 1 : -1))
    .map((message: Message) => ({
      senderName:
        users.value.get(message.sender.toHexString())?.name ||
        message.sender.toHexString().substring(0, 8),
      text: message.text,
    }));
});

const name = computed(() => {
  if (!identity.value) return 'unknown';
  return users.value.get(identity.value.toHexString())?.name || 
         identity.value.toHexString().substring(0, 8) || 
         'unknown';
});

// Store event handlers for cleanup
let messageInsertHandler: ((ctx: EventContext, message: Message) => void) | null = null;
let messageDeleteHandler: ((ctx: EventContext, message: Message) => void) | null = null;
let userInsertHandler: ((ctx: EventContext, user: User) => void) | null = null;
let userUpdateHandler: ((ctx: EventContext, oldUser: User, newUser: User) => void) | null = null;
let userDeleteHandler: ((ctx: EventContext, user: User) => void) | null = null;

onMounted(() => {
  const subscribeToQueries = (conn: DbConnection, queries: string[]) => {
    conn
      ?.subscriptionBuilder()
      .onApplied(() => {
        console.log('SDK client cache initialized.');
      })
      .subscribe(queries);
  };

  const onConnect = (
    connInstance: DbConnection,
    identityInstance: Identity,
    token: string
  ) => {
    identity.value = identityInstance;
    connected.value = true;
    localStorage.setItem('auth_token', token);
    console.log(
      'Connected to SpacetimeDB with identity:',
      identityInstance.toHexString()
    );
    
    connInstance.reducers.onSendMessage(() => {
      console.log('Message sent.');
    });

    // Set up database event listeners
    messageInsertHandler = (_ctx: EventContext, message: Message) => {
      messages.value = [...messages.value, message];
    };
    
    messageDeleteHandler = (_ctx: EventContext, message: Message) => {
      messages.value = messages.value.filter(
        m =>
          m.text !== message.text &&
          m.sent !== message.sent &&
          m.sender !== message.sender
      );
    };

    userInsertHandler = (_ctx: EventContext, user: User) => {
      users.value = new Map(users.value.set(user.identity.toHexString(), user));
    };

    userUpdateHandler = (_ctx: EventContext, oldUser: User, newUser: User) => {
      users.value = new Map(users.value.set(newUser.identity.toHexString(), newUser));
    };

    userDeleteHandler = (_ctx: EventContext, user: User) => {
      const newMap = new Map(users.value);
      newMap.delete(user.identity.toHexString());
      users.value = newMap;
    };

    // Register all the event handlers
    connInstance.db.message.onInsert(messageInsertHandler);
    connInstance.db.message.onDelete(messageDeleteHandler);
    connInstance.db.user.onInsert(userInsertHandler);
    connInstance.db.user.onUpdate(userUpdateHandler);
    connInstance.db.user.onDelete(userDeleteHandler);

    subscribeToQueries(connInstance, ['SELECT * FROM message', 'SELECT * FROM user']);
  };

  const onDisconnect = () => {
    console.log('Disconnected from SpacetimeDB');
    connected.value = false;
  };

  const onConnectError = (_ctx: ErrorContext, err: Error) => {
    console.log('Error connecting to SpacetimeDB:', err);
  };

  conn.value = (
    DbConnection.builder()
      .withUri('ws://localhost:3000')
      .withModuleName('quickstart-chat')
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
    if (messageInsertHandler) {
      conn.value.db.message.removeOnInsert(messageInsertHandler);
    }
    if (messageDeleteHandler) {
      conn.value.db.message.removeOnDelete(messageDeleteHandler);
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

const onSubmitNewName = () => {
  settingName.value = false;
  conn.value?.reducers.setName(nameFormState.name);
  nameFormState.name = '';
};

const onMessageSubmit = () => {
  conn.value?.reducers.sendMessage(messageFormState.message);
  messageFormState.message = '';
};
</script>

<template>
  <div v-if="connected" class="h-screen p-4">
    <div class="grid grid-cols-10 gap-4 h-full">
      <!-- Left column (70%) - Messages -->
      <div class="col-span-7 flex flex-col">
        <h1 class="text-2xl font-bold mb-4">Messages</h1>
        
        <!-- Messages container -->
        <div class="flex-1 overflow-y-auto space-y-3 mb-4">
          <p v-if="prettyMessages.length < 1" class="text-gray-500">No messages</p>
          
          <UCard v-for="(message, index) in prettyMessages" :key="index" class="mb-3">
            <template #header>
              <h3 class="font-semibold">{{ message.senderName }}</h3>
            </template>
            <p>{{ message.text }}</p>
          </UCard>
        </div>

        <!-- Message input form -->
        <UForm :state="messageFormState" @submit="onMessageSubmit" class="mt-auto">
          <div class="flex gap-2">
            <UInput 
              v-model="messageFormState.message" 
              placeholder="Type your message..." 
              class="flex-1"
            />
            <UButton type="submit">Send</UButton>
          </div>
        </UForm>
      </div>

      <!-- Right column (30%) - Profile and System -->
      <div class="col-span-3 space-y-6">
        <!-- Profile section -->
        <UCard>
          <template #header>
            <h2 class="text-xl font-semibold">Profile</h2>
          </template>
          
          <div v-if="!settingName">
            <p class="mb-3">{{ name }}</p>
            <UButton 
              @click="settingName = true; nameFormState.name = name" 
              variant="outline"
            >
              Edit Name
            </UButton>
          </div>
          
          <UForm v-else :state="nameFormState" @submit="onSubmitNewName">
            <div class="space-y-3">
              <UInput v-model="nameFormState.name" />
              <UButton type="submit">Submit</UButton>
            </div>
          </UForm>
        </UCard>

        <!-- System messages -->
        <UCard>
          <template #header>
            <h2 class="text-xl font-semibold">System</h2>
          </template>
          <pre class="whitespace-pre-wrap text-sm">{{ systemMessage }}</pre>
        </UCard>
      </div>
    </div>
  </div>
  
  <div v-else class="min-h-screen flex items-center justify-center">
    <p class="text-lg">Connecting to SpacetimeDB...</p>
  </div>
</template>