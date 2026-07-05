export function downloadFromUrl(url: string, filename?: string) {
  const anchor = document.createElement('a')
  anchor.href = url
  if (filename) {
    anchor.download = filename
  }
  anchor.click()
}

export function downloadBlob(blob: Blob, filename?: string) {
  const url = URL.createObjectURL(blob)
  try {
    downloadFromUrl(url, filename)
  } finally {
    URL.revokeObjectURL(url)
  }
}
