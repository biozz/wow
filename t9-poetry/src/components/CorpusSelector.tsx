import { For, createSignal } from 'solid-js'
import type { CorpusFile } from '../utils/corpusLoader'
import { loadCorpusFromUrl } from '../utils/corpusLoader'

interface CorpusSelectorProps {
  corpusFiles: CorpusFile[]
  selectedCorpus: CorpusFile | null
  onCorpusSelect: (corpus: CorpusFile) => void
}

export default function CorpusSelector(props: CorpusSelectorProps) {
  const [urlInput, setUrlInput] = createSignal('')
  const [urlLoading, setUrlLoading] = createSignal(false)
  const [urlError, setUrlError] = createSignal('')

  const handleUrlLoad = async () => {
    const url = urlInput().trim()
    if (!url) return

    setUrlLoading(true)
    setUrlError('')

    try {
      const corpus = await loadCorpusFromUrl(url)
      props.onCorpusSelect(corpus)
      setUrlInput('')
    } catch (error) {
      setUrlError(error instanceof Error ? error.message : 'Ошибка загрузки')
    } finally {
      setUrlLoading(false)
    }
  }

  const handleKeyPress = (e: KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleUrlLoad()
    }
  }

  return (
    <div class="bg-white rounded-xl shadow-lg p-6 mb-6">
      <h3 class="text-xl font-semibold text-gray-800 mb-4 text-center">
        Выберите корпус текста:
      </h3>
      
      {/* URL Input Section */}
      <div class="mb-6 p-4 bg-gray-50 rounded-lg">
        <h4 class="text-lg font-medium text-gray-700 mb-3">Или загрузите из URL:</h4>
        <div class="flex gap-2">
          <input
            type="url"
            value={urlInput()}
            onInput={(e) => setUrlInput(e.currentTarget.value)}
            onKeyPress={handleKeyPress}
            placeholder="https://example.com/text.txt"
            class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            disabled={urlLoading()}
          />
          <button
            onClick={handleUrlLoad}
            disabled={urlLoading() || !urlInput().trim()}
            class="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {urlLoading() ? (
              <>
                <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                Загрузка...
              </>
            ) : (
              'Загрузить'
            )}
          </button>
        </div>
        {urlError() && (
          <div class="mt-2 text-sm text-red-600">
            {urlError()}
          </div>
        )}
      </div>

      {/* Built-in Corpus Selection */}
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <For each={props.corpusFiles}>
          {(corpus) => (
            <button
              onClick={() => props.onCorpusSelect(corpus)}
              class={`p-4 rounded-lg border-2 transition-all duration-200 ${
                props.selectedCorpus?.filename === corpus.filename
                  ? 'border-blue-500 bg-blue-50 text-blue-700'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
              }`}
            >
              <div class="font-medium capitalize">{corpus.name}</div>
              <div class="text-sm text-gray-500 mt-1">
                {corpus.words.length} слов
              </div>
            </button>
          )}
        </For>
      </div>
    </div>
  )
}
