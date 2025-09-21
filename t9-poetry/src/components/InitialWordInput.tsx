import { createSignal, Show } from 'solid-js'

interface InitialWordInputProps {
  onWordSet: (word: string) => void
}

export default function InitialWordInput(props: InitialWordInputProps) {
  const [inputValue, setInputValue] = createSignal('')
  const [isSet, setIsSet] = createSignal(false)
  const [setWord, setSetWord] = createSignal('')

  const handleSubmit = (e: Event) => {
    e.preventDefault()
    const word = inputValue().trim()
    if (word) {
      setSetWord(word)
      setIsSet(true)
      props.onWordSet(word)
    }
  }

  return (
    <div class="bg-white rounded-xl shadow-lg p-6 mb-6">
      <Show when={!isSet()}>
        <form onSubmit={handleSubmit} class="max-w-md mx-auto">
          <div class="flex gap-2">
            <input
              type="text"
              value={inputValue()}
              onInput={(e) => setInputValue(e.currentTarget.value)}
              placeholder="В начале было слово..."
              class="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              required
            />
            <button
              type="submit"
              class="btn-primary px-6 py-2"
            >
              Установить
            </button>
          </div>
        </form>
      </Show>
    </div>
  )
}
