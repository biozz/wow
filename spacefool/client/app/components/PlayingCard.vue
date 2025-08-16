<script setup lang="ts">
import { computed } from 'vue';
import type { Card, Rank, Suit } from '../../module_bindings';

interface Props {
  card: Card;
  selected?: boolean;
  disabled?: boolean;
}

const props = defineProps<Props>();
const emit = defineEmits<{ (e: 'click', card: Card): void }>();

const rankText = computed(() => props.card.rank.tag.replace('Jack','J').replace('Queen','Q').replace('King','K').replace('Ace','A').replace('Ten','10').replace('Nine','9').replace('Eight','8').replace('Seven','7').replace('Six','6'));
const suitSymbol = computed(() => {
  switch (props.card.suit.tag) {
    case 'Hearts': return '♥';
    case 'Diamonds': return '♦';
    case 'Clubs': return '♣';
    case 'Spades': return '♠';
  }
});
const isRed = computed(() => ['Hearts','Diamonds'].includes(props.card.suit.tag));

const onClick = () => {
  if (!props.disabled) emit('click', props.card);
};
</script>

<template>
  <button
    type="button"
    class="relative w-16 h-24 rounded-md border transition transform hover:-translate-y-1"
    :class="[
      selected ? 'ring-2 ring-primary border-primary' : 'border-gray-300',
      disabled ? 'opacity-50 cursor-not-allowed' : 'bg-white'
    ]"
    @click="onClick"
  >
    <div class="absolute top-1 left-1 text-xs" :class="isRed ? 'text-red-600' : 'text-gray-900'">
      {{ rankText }}
      <span class="ml-0.5">{{ suitSymbol }}</span>
    </div>
    <div class="flex items-center justify-center h-full text-2xl" :class="isRed ? 'text-red-600' : 'text-gray-900'">
      {{ suitSymbol }}
    </div>
    <div class="absolute bottom-1 right-1 text-xs rotate-180" :class="isRed ? 'text-red-600' : 'text-gray-900'">
      {{ rankText }}
      <span class="ml-0.5">{{ suitSymbol }}</span>
    </div>
  </button>
</template>
