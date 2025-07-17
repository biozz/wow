// https://nuxt.com/docs/api/configuration/nuxt-config
import { defineNuxtConfig } from 'nuxt/config'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  imports: {
    autoImport: false,
  },
  modules: ['@nuxt/eslint', '@nuxt/ui'],
  devServer: {
    port: 3001,
  },
  css: ['~/assets/css/main.css'],
})