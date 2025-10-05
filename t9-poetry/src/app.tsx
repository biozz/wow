import { createSignal, For, onMount, Show } from 'solid-js'
import { loadAllCorpusFiles, type CorpusFile } from './utils/corpusLoader'
import CorpusSelector from './components/CorpusSelector'
import InitialWordInput from './components/InitialWordInput'

// Get random word suggestions
function getRandomSuggestions(words: string[]): string[] {
  return words
    .sort(() => Math.random() - 0.5)
    .slice(0, 3)
}

export default function App() {
  const [selectedWords, setSelectedWords] = createSignal<string[]>([])
  const [suggestions, setSuggestions] = createSignal<string[]>([])
  const [rerollsLeft, setRerollsLeft] = createSignal(3)
  const [corpusFiles, setCorpusFiles] = createSignal<CorpusFile[]>([])
  const [selectedCorpus, setSelectedCorpus] = createSignal<CorpusFile | null>(null)
  const [loading, setLoading] = createSignal(true)
  const [corpusSelected, setCorpusSelected] = createSignal(false)
  const [initialWord, setInitialWord] = createSignal<string>('')
  const [initialWordSet, setInitialWordSet] = createSignal(false)

  // Load corpus files on mount
  onMount(async () => {
    try {
      const files = await loadAllCorpusFiles()
      setCorpusFiles(files)
      // Don't auto-select first corpus - let user choose
    } catch (error) {
      console.error('Failed to load corpus files:', error)
    } finally {
      setLoading(false)
    }
  })

  // Handle initial word setting
  const handleInitialWordSet = (word: string) => {
    setInitialWord(word)
    setInitialWordSet(true)
    setSelectedWords([word]) // Set the initial word as the first selected word
  }

  // Handle corpus selection
  const handleCorpusSelect = (corpus: CorpusFile) => {
    setSelectedCorpus(corpus)
    setSuggestions(getRandomSuggestions(corpus.words))
    // Don't clear selected words if initial word is set - keep it
    if (initialWordSet()) {
      setSelectedWords([initialWord()])
    } else {
      setSelectedWords([])
    }
    setRerollsLeft(3) // Reset rerolls
    setCorpusSelected(true) // Mark corpus as selected
  }

  // Handle word selection
  const selectWord = (word: string) => {
    setSelectedWords(prev => [...prev, word])
    // Generate new suggestions after selection
    const currentCorpus = selectedCorpus()
    if (currentCorpus) {
      setSuggestions(getRandomSuggestions(currentCorpus.words))
    }
  }

  // Clear all
  const clearAll = () => {
    // Keep initial word if it's set
    if (initialWordSet()) {
      setSelectedWords([initialWord()])
    } else {
      setSelectedWords([])
    }
    const currentCorpus = selectedCorpus()
    if (currentCorpus) {
      setSuggestions(getRandomSuggestions(currentCorpus.words))
    }
    setRerollsLeft(3)
  }

  // Remove last word
  const removeLastWord = () => {
    setSelectedWords(prev => {
      const newWords = prev.slice(0, -1)
      // Don't allow removing the initial word if it's the only word left
      if (initialWordSet() && newWords.length === 0) {
        return [initialWord()]
      }
      return newWords
    })
    const currentCorpus = selectedCorpus()
    if (currentCorpus) {
      setSuggestions(getRandomSuggestions(currentCorpus.words))
    }
  }

  // Generate new suggestions (reroll)
  const reroll = () => {
    if (rerollsLeft() > 0) {
      const currentCorpus = selectedCorpus()
      if (currentCorpus) {
        setSuggestions(getRandomSuggestions(currentCorpus.words))
      }
      setRerollsLeft(prev => prev - 1)
    }
  }

  // Add newline to poetry
  const addNewline = () => {
    setSelectedWords(prev => [...prev, '\n'])
  }

  // Copy result to clipboard
  const copyResult = async () => {
    const text = selectedWords().join(' ')
    try {
      await navigator.clipboard.writeText(text)
      // You could add a toast notification here if desired
    } catch (err) {
      console.error('Failed to copy text: ', err)
    }
  }

  return (
    <div class="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-8">
      <div class="max-w-4xl mx-auto">
        <header class="text-center mb-8">
          <h1 class="text-4xl font-bold text-gray-800 mb-2">Поэтический костыль</h1>
          <Show when={initialWordSet()}>
            <div class="text-sm text-gray-600 space-y-1">
              <Show when={corpusSelected()}>
                <div>Корпус: <span class="text-blue-600 capitalize">{selectedCorpus()?.name}</span></div>
              </Show>
              <div>Начальное слово: <span class="text-green-600">{initialWord()}</span></div>
            </div>
          </Show>
        </header>

        <Show when={!loading()} fallback={
          <div class="bg-white rounded-xl shadow-lg p-8 mb-6 text-center">
            <div class="flex flex-col items-center space-y-4">
              <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
              <div class="text-gray-600">Загрузка корпусов текста...</div>
            </div>
          </div>
        }>
          {/* Initial Word Input - only show if not set yet */}
          <Show when={!initialWordSet()}>
            <InitialWordInput onWordSet={handleInitialWordSet} />
          </Show>

          {/* Corpus Selection - only show if not selected yet */}
          <Show when={!corpusSelected()}>
            <CorpusSelector
              corpusFiles={corpusFiles()}
              selectedCorpus={selectedCorpus()}
              onCorpusSelect={handleCorpusSelect}
            />
          </Show>

          {/* Poetry and Word Selection - only show if corpus is selected and initial word is set */}
          <Show when={corpusSelected() && initialWordSet()}>
            <div class="bg-white rounded-xl shadow-lg p-8 mb-6">

              {/* Poetry action buttons */}
              <div class="flex gap-2 justify-center mb-6 flex-wrap">
                <button
                  onClick={reroll}
                  disabled={rerollsLeft() === 0}
                  class="btn-secondary text-sm disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Другие слова ({rerollsLeft()})
                </button>
                <button
                  onClick={addNewline}
                  class="btn-secondary text-sm"
                >
                  Новая строка
                </button>
                <Show when={selectedWords().length > 0}>
                  <button
                    onClick={removeLastWord}
                    class="btn-secondary text-sm"
                  >
                    Удалить последнее
                  </button>
                  <button
                    onClick={copyResult}
                    class="btn-primary text-sm"
                  >
                    Копировать результат
                  </button>
                </Show>
              </div>
              {/* Word selection buttons */}
              <div class="text-center mb-6">
                <div class="flex gap-4 justify-center mb-4">
                  <For each={suggestions()}>
                    {(word) => (
                      <button
                        onClick={() => selectWord(word)}
                        class="btn-primary text-lg px-8 py-4 min-w-32"
                      >
                        {word}
                      </button>
                    )}
                  </For>
                </div>
              </div>

              {/* Poetry result */}
              <Show when={selectedWords().length > 0}>
                <div class="text-xl text-gray-800 leading-relaxed whitespace-pre-line text-center">
                  {selectedWords().join(' ')}
                </div>
              </Show>
            </div>
          </Show>

        </Show>

      </div>
    </div>
  )
}
