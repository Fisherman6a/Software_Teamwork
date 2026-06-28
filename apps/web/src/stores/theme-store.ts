import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ThemeState = {
  mode: 'light' | 'dark'
  setMode: (mode: 'light' | 'dark') => void
  primaryColor: string
  setPrimaryColor: (color: string) => void
  radius: number
  setRadius: (radius: number) => void
  fontSize: 'small' | 'medium' | 'large'
  setFontSize: (size: 'small' | 'medium' | 'large') => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      mode: 'light',
      setMode: (mode) => set({ mode }),
      primaryColor: 'neutral',
      setPrimaryColor: (primaryColor) => set({ primaryColor }),
      radius: 0.625,
      setRadius: (radius) => set({ radius }),
      fontSize: 'medium',
      setFontSize: (fontSize) => set({ fontSize }),
    }),
    {
      name: 'qa-theme-preferences',
    },
  ),
)
