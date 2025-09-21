// Utility to load and process text files into word arrays
export interface CorpusFile {
  name: string
  filename: string
  words: string[]
}

// Remove punctuation and split text into words
function processText(text: string): string[] {
  const words = text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s]/gu, ' ') // Replace punctuation with spaces (supports Unicode letters and numbers)
    .split(/\s+/) // Split on whitespace
    .filter(word => word.length > 0) // Remove empty strings
  
  // Remove duplicates using Set
  return Array.from(new Set(words))
}

// Load a text file and convert to word array
export async function loadCorpusFile(filename: string): Promise<CorpusFile> {
  try {
    const response = await fetch(`/${filename}`)
    if (!response.ok) {
      throw new Error(`Failed to load ${filename}`)
    }
    const text = await response.text()
    const words = processText(text)
    return {
      name: filename.replace('.txt', '').replace(/([A-Z])/g, ' $1').trim(),
      filename,
      words
    }
  } catch (error) {
    console.error(`Error loading corpus file ${filename}:`, error)
    throw error
  }
}

// Load text from URL and convert to word array
export async function loadCorpusFromUrl(url: string): Promise<CorpusFile> {
  try {
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`Failed to load text from URL: ${response.status} ${response.statusText}`)
    }
    const text = await response.text()
    const words = processText(text)
    return {
      name: 'Другой',
      filename: url,
      words
    }
  } catch (error) {
    console.error(`Error loading corpus from URL ${url}:`, error)
    throw error
  }
}

// Get list of available corpus files
export async function getAvailableCorpusFiles(): Promise<string[]> {
  // For now, return hardcoded list. In a real app, you might fetch this from an API
  return ['poetry.txt', 'classic.txt', 'modern.txt', 'karenina.txt']
}

// Load all available corpus files
export async function loadAllCorpusFiles(): Promise<CorpusFile[]> {
  const filenames = await getAvailableCorpusFiles()
  const corpusFiles = await Promise.all(
    filenames.map(filename => loadCorpusFile(filename))
  )
  return corpusFiles
}
