import { describe, expect, it, vi } from 'vitest'

import { downloadBlob, downloadFromUrl } from './download'

describe('downloadFromUrl', () => {
  it('creates a temporary anchor and triggers a browser download', () => {
    const click = vi.fn()
    const createElement = vi.spyOn(document, 'createElement')

    createElement.mockImplementation((tagName: string) => {
      const element = document.createElementNS('http://www.w3.org/1999/xhtml', tagName)
      if (tagName === 'a') {
        Object.defineProperty(element, 'click', { configurable: true, value: click })
      }
      return element as HTMLElement
    })

    downloadFromUrl('/api/v1/report-files/file-1/content', 'report.docx')

    const anchor = createElement.mock.results[0]?.value as HTMLAnchorElement
    expect(anchor.href).toContain('/api/v1/report-files/file-1/content')
    expect(anchor.download).toBe('report.docx')
    expect(click).toHaveBeenCalledTimes(1)
  })
})

describe('downloadBlob', () => {
  it('creates and revokes an object url after triggering the download', () => {
    const click = vi.fn()
    const createElement = vi.spyOn(document, 'createElement')
    const createObjectURL = vi.spyOn(URL, 'createObjectURL')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL')

    createElement.mockImplementation((tagName: string) => {
      const element = document.createElementNS('http://www.w3.org/1999/xhtml', tagName)
      if (tagName === 'a') {
        Object.defineProperty(element, 'click', { configurable: true, value: click })
      }
      return element as HTMLElement
    })
    createObjectURL.mockReturnValue('blob:report-file')
    revokeObjectURL.mockImplementation(() => undefined)

    const blob = new Blob(['docx'])
    downloadBlob(blob, 'report.docx')

    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalledTimes(1)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:report-file')
  })

  it('revokes the object url when the browser download fails', () => {
    const createElement = vi.spyOn(document, 'createElement')
    const createObjectURL = vi.spyOn(URL, 'createObjectURL')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL')

    createElement.mockImplementation((tagName: string) => {
      const element = document.createElementNS('http://www.w3.org/1999/xhtml', tagName)
      if (tagName === 'a') {
        Object.defineProperty(element, 'click', {
          configurable: true,
          value: () => {
            throw new Error('download blocked')
          },
        })
      }
      return element as HTMLElement
    })
    createObjectURL.mockReturnValue('blob:report-file')
    revokeObjectURL.mockImplementation(() => undefined)

    expect(() => downloadBlob(new Blob(['docx']), 'report.docx')).toThrow('download blocked')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:report-file')
  })
})
