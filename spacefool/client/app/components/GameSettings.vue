<script setup lang="ts">
import { computed, reactive } from 'vue';
import { GameSettings, DeckSize } from '../../module_bindings';

interface Props {
  settings: GameSettings;
  isCreator: boolean;
  onUpdate: (settings: Partial<GameSettings>) => void;
}

const props = defineProps<Props>();

const deckSizeOptions = [
  { value: 'Standard36', label: 'Standard (36 cards)' },
  { value: 'Extended52', label: 'Extended (52 cards)' }
];

const formState = reactive({
  deckSize: props.settings.deckSize.tag,
  startingCards: props.settings.startingCards,
  maxAttackCards: props.settings.maxAttackCards,
  multiRoundMode: props.settings.multiRoundMode,
  maxPoints: props.settings.maxPoints,
  anyoneCanAttack: props.settings.anyoneCanAttack,
  trumpCardToPlayer: props.settings.trumpCardToPlayer
});

const handleSubmit = () => {
  const deckSize: DeckSize = formState.deckSize === 'Standard36' 
    ? { tag: 'Standard36' as const } 
    : { tag: 'Extended52' as const };
  const { deckSize: _, ...rest } = formState;
  props.onUpdate({
    ...rest,
    deckSize
  });
};
</script>

<template>
  <UCard>
    <template #header>
      <h3 class="text-lg font-semibold">Game Settings</h3>
    </template>

    <UForm :state="formState" @submit="handleSubmit">
      <div class="space-y-4">
        <!-- Deck Size -->
        <UFormGroup label="Deck Size" name="deckSize">
          <USelect
            v-model="formState.deckSize"
            :options="deckSizeOptions"
            option-attribute="label"
            value-attribute="value"
            :disabled="!isCreator"
          />
        </UFormGroup>

        <!-- Starting Cards -->
        <UFormGroup label="Starting Cards" name="startingCards">
          <UInputNumber
            v-model="formState.startingCards"
            :min="3"
            :max="20"
            :disabled="!isCreator"
          />
        </UFormGroup>

        <!-- Max Attack Cards -->
        <UFormGroup label="Max Attack Cards (0 = no limit)" name="maxAttackCards">
          <UInputNumber
            v-model="formState.maxAttackCards"
            :min="0"
            :max="12"
            :disabled="!isCreator"
          />
        </UFormGroup>

        <!-- Multi Round Mode -->
        <UFormGroup label="Multi Round Mode" name="multiRoundMode">
          <UCheckbox
            v-model="formState.multiRoundMode"
            :disabled="!isCreator"
          />
          <template #label>
            Traditional Durak with point system
          </template>
        </UFormGroup>

        <!-- Max Points (only show if multi round) -->
        <UFormGroup 
          v-if="formState.multiRoundMode" 
          label="Max Points" 
          name="maxPoints"
        >
          <UInputNumber
            v-model="formState.maxPoints"
            :min="5"
            :max="50"
            :disabled="!isCreator"
          />
        </UFormGroup>

        <!-- Anyone Can Attack -->
        <UFormGroup label="Anyone Can Attack" name="anyoneCanAttack">
          <UCheckbox
            v-model="formState.anyoneCanAttack"
            :disabled="!isCreator"
          />
          <template #label>
            Any player can join attacks (traditional rule)
          </template>
        </UFormGroup>

        <!-- Trump Card To Player -->
        <UFormGroup label="Trump Card To Player" name="trumpCardToPlayer">
          <UCheckbox
            v-model="formState.trumpCardToPlayer"
            :disabled="!isCreator"
          />
          <template #label>
            Trump card goes to last dealt player
          </template>
        </UFormGroup>
      </div>

      <div class="flex justify-end mt-4">
        <UButton
          v-if="isCreator"
          type="submit"
          color="primary"
        >
          Update Settings
        </UButton>
      </div>
    </UForm>
  </UCard>
</template> 