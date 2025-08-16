<script setup lang="ts">
import { shallowRef, computed, onMounted, onUnmounted, reactive } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { DbConnection, type ReducerEventContext, type SubscriptionEventContext, Card, Game, User, Turn, Draw, PlayerCard, CardLocation, Round } from '../../../module_bindings';
import type { Identity } from '@clockworklabs/spacetimedb-sdk';

const route = useRoute();
const router = useRouter();
const gameId = computed(() => BigInt(route.params.id as string));

// Connection
const connected = shallowRef<boolean>(false);
const identity = shallowRef<Identity | null>(null);
const conn = shallowRef<DbConnection | null>(null);
// Hidden CPU connection (if cpu_mode)
const botConn = shallowRef<DbConnection | null>(null);
const cpuBotIdHex = shallowRef<string | null>(typeof localStorage !== 'undefined' ? localStorage.getItem('cpu_bot_id') : null);

// Cache
const game = shallowRef<Game | null>(null);
const users = shallowRef<Map<string, User>>(new Map());
const currentUser = shallowRef<User | null>(null);
const turns = shallowRef<Turn[]>([]);
const draws = shallowRef<Draw[]>([]);
const playerCards = shallowRef<PlayerCard[]>([]);
const rounds = shallowRef<Round[]>([]);

// UI state
const selectedCard = shallowRef<Card | null>(null);
const selectedDrawId = shallowRef<bigint | null>(null);
const selectedTargetHex = shallowRef<string | null>(null);

const myRounds = computed(() => rounds.value.filter(r => r.gameId === gameId.value));
const myRoundIds = computed(() => new Set(myRounds.value.map(r => r.id)));
const myTurns = computed(() => turns.value.filter(t => myRoundIds.value.has(t.roundId)));
const currentTurn = computed(() => myTurns.value
  .filter(t => t.status.tag === 'Active')
  .sort((a,b) => Number(a.turnNumber - b.turnNumber))[0]
);

const playerCardsInGame = computed(() => playerCards.value.filter(pc => pc.gameId === gameId.value));
const deckCount = computed(() => playerCardsInGame.value.filter(pc => pc.location.tag === 'Deck').length);
const myHand = computed(() => playerCardsInGame.value.filter(pc => pc.player.toHexString() === identity.value?.toHexString() && pc.location.tag === 'Hand').map(pc => pc.card));
const tableCards = computed(() => playerCardsInGame.value.filter(pc => pc.location.tag === 'OnTable'));
const tableDraws = computed(() => {
  const t = currentTurn.value;
  if (!t) return [] as Draw[];
  return draws.value.filter(d => d.turnId === t.id);
});

const defenderId = computed(() => currentTurn.value?.defender);
const isDefender = computed(() => defenderId.value && currentUser.value && defenderId.value.toHexString() === currentUser.value.identity.toHexString());
// CPU control runs if cpu_mode and we have a bot connection and bot user resolved
const isCpuEnabled = computed(() => !!localStorage.getItem('cpu_mode') && !!botConn.value && !!cpuBotIdHex.value);

// Actions
const callAttack = (card: Card) => {
  if (!conn.value) return;
  if (currentTurn.value) {
    // Attack current defender
    conn.value.reducers.attack(gameId.value, card, currentTurn.value.defender);
  } else {
    // Start a new turn: require a selected target
    const target = otherPlayers.value.find(p => p.identity.toHexString() === selectedTargetHex.value)?.identity;
    if (!target) return;
    conn.value.reducers.attack(gameId.value, card, target);
  }
  selectedCard.value = null;
};

const callDefend = () => {
  if (!conn.value || !currentTurn.value || !selectedCard.value) return;
  conn.value.reducers.defend(gameId.value, currentTurn.value.id, selectedCard.value);
  selectedCard.value = null;
  selectedDrawId.value = null;
};

const callTake = () => {
  if (!conn.value || !currentTurn.value) return;
  conn.value.reducers.takeCards(gameId.value, currentTurn.value.id);
};

const callPass = () => {
  if (!conn.value) return;
  conn.value.reducers.passTurn(gameId.value);
};

const gamePlayers = computed(() => Array.from(users.value.values()).filter(u => u.currentGameId === gameId.value));
const otherPlayers = computed(() => gamePlayers.value.filter(u => u.identity.toHexString() !== currentUser.value?.identity.toHexString()));
const botUser = computed(() => Array.from(users.value.values()).find(u => u.identity.toHexString() === cpuBotIdHex.value));
const otherPlayersForBot = computed(() => gamePlayers.value.filter(u => u.identity.toHexString() !== cpuBotIdHex.value));
const botHand = computed(() => playerCardsInGame.value.filter(pc => pc.player.toHexString() === cpuBotIdHex.value && pc.location.tag === 'Hand').map(pc => pc.card));

// --- Basic CPU logic (only when this browser session is the CPU tab) ---
// Heuristic:
// - If defender: try to defend with minimal winning card, else take
// - If attacker and no active turn: attack lowest-rank card targeting next player
const rankOrder = ['Six','Seven','Eight','Nine','Ten','Jack','Queen','King','Ace'];
function cardCmp(a: Card, b: Card): number {
  const ar = rankOrder.indexOf(a.rank.tag);
  const br = rankOrder.indexOf(b.rank.tag);
  if (a.suit.tag === game.value?.trumpSuit.tag && b.suit.tag !== game.value?.trumpSuit.tag) return 1;
  if (a.suit.tag !== game.value?.trumpSuit.tag && b.suit.tag === game.value?.trumpSuit.tag) return -1;
  if (ar === br) return 0;
  return ar - br;
}

function canBeat(attacking: Card, defending: Card): boolean {
  const trump = game.value?.trumpSuit.tag;
  const atkTrump = attacking.suit.tag === trump;
  const defTrump = defending.suit.tag === trump;
  if (atkTrump && defTrump) return rankOrder.indexOf(defending.rank.tag) > rankOrder.indexOf(attacking.rank.tag);
  if (!atkTrump && defTrump) return true;
  if (atkTrump && !defTrump) return false;
  return attacking.suit.tag === defending.suit.tag && rankOrder.indexOf(defending.rank.tag) > rankOrder.indexOf(attacking.rank.tag);
}

function pickDefense(attacks: Draw[], hand: Card[]): Card | null {
  // defend the first pending with minimal winning card
  const pending = attacks.find(d => d.status.tag === 'Pending');
  if (!pending) return null;
  const candidates = hand.filter(c => canBeat(pending.attackingCard, c)).sort(cardCmp);
  return candidates[0] || null;
}

function pickAttack(hand: Card[], ranksOnTable: Set<string>): Card | null {
  const sorted = [...hand].sort(cardCmp);
  if (ranksOnTable.size === 0) return sorted[0] || null;
  for (const c of sorted) if (ranksOnTable.has(c.rank.tag)) return c;
  return null;
}

function ranksFromDraws(ds: Draw[]): Set<string> {
  const s = new Set<string>();
  for (const d of ds) { s.add(d.attackingCard.rank.tag); if (d.defendingCard) s.add(d.defendingCard.rank.tag); }
  return s;
}

onMounted(() => {
  const onConnect = (connInstance: DbConnection, identityInstance: Identity, token: string) => {
    identity.value = identityInstance;
    connected.value = true;
    localStorage.setItem('auth_token', token);

    // Table events
    connInstance.db.game.onUpdate((_ctx, _oldRow, newRow) => {
      if (newRow.id === gameId.value) game.value = newRow;
    });

    connInstance.db.user.onInsert((_ctx, row) => {
      users.value = new Map(users.value.set(row.identity.toHexString(), row));
      if (row.identity.toHexString() === identityInstance.toHexString()) currentUser.value = row;
    });
    connInstance.db.user.onUpdate((_ctx, _old, row) => {
      users.value = new Map(users.value.set(row.identity.toHexString(), row));
      if (row.identity.toHexString() === identityInstance.toHexString()) currentUser.value = row;
    });

    connInstance.db.turn.onInsert((_ctx, row) => {
      turns.value = [...turns.value, row];
    });
    connInstance.db.turn.onUpdate((_ctx, _old, row) => {
      const idx = turns.value.findIndex(t => t.id === row.id);
      if (idx >= 0) turns.value.splice(idx, 1, row);
      else turns.value = [...turns.value, row];
    });

    connInstance.db.draw.onInsert((_ctx, row) => {
      draws.value = [...draws.value, row];
    });
    connInstance.db.draw.onUpdate((_ctx, _old, row) => {
      const idx = draws.value.findIndex(d => d.id === row.id);
      if (idx >= 0) draws.value.splice(idx, 1, row);
      else draws.value = [...draws.value, row];
    });

    connInstance.db.playerCard.onInsert((_ctx, row) => {
      playerCards.value = [...playerCards.value, row];
    });
    connInstance.db.playerCard.onUpdate((_ctx, _old, row) => {
      const idx = playerCards.value.findIndex(pc => pc.id === row.id);
      if (idx >= 0) playerCards.value.splice(idx, 1, row);
      else playerCards.value = [...playerCards.value, row];
    });

    connInstance.db.round.onInsert((_ctx, row) => { rounds.value = [...rounds.value, row]; });
    connInstance.db.round.onUpdate((_ctx, _old, row) => {
      const idx = rounds.value.findIndex(r => r.id === row.id);
      if (idx >= 0) rounds.value.splice(idx, 1, row); else rounds.value = [...rounds.value, row];
    });

    // Subscribe
    connInstance
      .subscriptionBuilder()
      .onApplied(() => {
        // noop
      })
      .subscribe([
        `SELECT * FROM game WHERE id = ${gameId.value}`,
        'SELECT * FROM user',
        'SELECT * FROM round',
        'SELECT * FROM turn',
        'SELECT * FROM draw',
        'SELECT * FROM player_card',
      ]);
  };

  const onDisconnect = () => {
    connected.value = false;
  };

  const onConnectError = (_ctx: any, err: Error) => {
    console.error('Error connecting to SpacetimeDB:', err);
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
  // Bring up bot connection if cpu_mode
  if (localStorage.getItem('cpu_mode') === '1') {
    const onBotConnect = (c: DbConnection, id: Identity, token: string) => {
      // Persist bot id
      cpuBotIdHex.value = id.toHexString();
      try { localStorage.setItem('cpu_bot_id', cpuBotIdHex.value!); } catch (_) {}
      try { localStorage.setItem('cpu_bot_token', token); } catch (_) {}
      // Name it (best-effort)
      try { c.reducers.setName('CPU'); } catch (_) {}
    };
    const onBotDisconnect = () => {};
    const onBotError = (_ctx: any, _err: Error) => {};
    botConn.value = (
      DbConnection.builder()
        .withUri('ws://localhost:3000')
        .withModuleName('spacefool')
        .onConnect(onBotConnect)
        .onDisconnect(onBotDisconnect)
        .onConnectError(onBotError)
        .build()
    );
  }

  // CPU driver loop: act via botConn using shared state from main connection
  const tick = async () => {
    try {
      if (!isCpuEnabled.value || !game.value || !botUser.value || !botConn.value) return;
      const hand = botHand.value;
      if (!hand.length) return;
      // If defender and there is a pending attack, try defend or take
      const botIsDefender = currentTurn.value && defenderId.value && defenderId.value.toHexString() === cpuBotIdHex.value;
      if (botIsDefender && currentTurn.value) {
        const def = pickDefense(tableDraws.value, hand);
        if (def) {
          botConn.value.reducers.defend(gameId.value, currentTurn.value.id, def);
        } else {
          botConn.value.reducers.takeCards(gameId.value, currentTurn.value.id);
        }
      } else {
        // If no active turn, try to initiate an attack at next player
        if (!currentTurn.value && otherPlayersForBot.value.length) {
          const targetUser = otherPlayersForBot.value[0];
          if (!targetUser) return;
          const target = targetUser.identity;
          const rs = ranksFromDraws(tableDraws.value);
          const atk = pickAttack(hand, rs);
          if (atk) botConn.value.reducers.attack(gameId.value, atk, target);
        }
      }
    } catch (_) {}
  };
  const interval = setInterval(tick, 350);
  onUnmounted(() => clearInterval(interval));
});

onUnmounted(() => {
  // minimal cleanup: page unmounts
});
</script>

<template>
  <div v-if="connected && game" class="min-h-screen p-4">
    <div class="max-w-6xl mx-auto space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Game #{{ game.id.toString() }}</h1>
          <p class="text-sm text-gray-500">Trump: {{ game.trumpSuit.tag }}</p>
        </div>
        <div class="flex gap-2">
          <UBadge :color="game.status.tag === 'Active' ? 'success' : 'neutral'" variant="subtle">{{ game.status.tag }}</UBadge>
          <UBadge variant="subtle">Deck: {{ deckCount }}</UBadge>
        </div>
      </div>

      <!-- Table -->
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold">Table</h2>
            <div v-if="currentTurn">
              <span class="text-sm text-gray-600">Turn #{{ currentTurn.turnNumber }}</span>
            </div>
          </div>
        </template>

        <div class="grid grid-cols-2 gap-6">
          <div>
            <h3 class="font-medium mb-2">Attacks & Defenses</h3>
            <div class="flex flex-wrap gap-3">
              <div v-for="draw in tableDraws" :key="draw.id.toString()" class="flex items-center gap-2">
                <PlayingCard :card="draw.attackingCard" />
                <UIcon name="i-lucide-arrow-right" class="text-gray-400" />
                <PlayingCard v-if="draw.defendingCard" :card="draw.defendingCard" />
                <UBadge v-else color="warning" variant="subtle" size="xs">Pending</UBadge>
              </div>
              <div v-if="tableDraws.length === 0" class="text-gray-500 text-sm">No cards on table</div>
            </div>
          </div>
          <div>
            <h3 class="font-medium mb-2">Actions</h3>
            <div class="flex flex-wrap gap-2 items-center">
              <USelect
                v-if="!currentTurn"
                v-model="selectedTargetHex"
                :options="otherPlayers.map(p => ({ label: p.name || p.identity.toHexString().substring(0,8), value: p.identity.toHexString() }))"
                placeholder="Select defender"
                size="sm"
                class="min-w-40"
              />
              <UButton color="primary" :disabled="!selectedCard" @click="selectedCard && callAttack(selectedCard)">Attack</UButton>
              <UButton color="success" :disabled="!selectedCard || !isDefender" @click="callDefend">Defend</UButton>
              <UButton color="error" variant="outline" :disabled="!isDefender" @click="callTake">Take</UButton>
              <UButton variant="outline" @click="callPass">Pass</UButton>
              <UButton variant="ghost" @click="router.push('/lobbies')">Leave</UButton>
            </div>
            <p class="text-xs text-gray-500 mt-2">Select a card from your hand to Attack or Defend.</p>
          </div>
        </div>
      </UCard>

      <!-- Hand -->
      <UCard>
        <template #header>
          <h2 class="text-lg font-semibold">Your Hand</h2>
        </template>
        <div class="flex flex-wrap gap-2">
          <PlayingCard
            v-for="(card, idx) in myHand"
            :key="idx"
            :card="card"
            :selected="!!selectedCard && selectedCard.suit.tag === card.suit.tag && selectedCard.rank.tag === card.rank.tag"
            @click="selectedCard = card"
          />
          <div v-if="myHand.length === 0" class="text-gray-500 text-sm">No cards</div>
        </div>
      </UCard>
    </div>
  </div>
  <div v-else class="min-h-screen flex items-center justify-center">
    <p class="text-lg">Connecting to game...</p>
  </div>
</template>

<style scoped>
</style>
