import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { ConfigManager } from '../../bindings/app/backend'
import { toast } from 'vue-sonner'
import { reportError } from '../utils/errors'

export const selectedAccount = ref('')
export const accountOptions = ref<string[]>([])
const credentialMap = ref<Record<string, string>>({})

export const tokenBindingEmail = ref('')
export const isExtractingTokenBinding = ref(false)

function extractEmailFromCredential(credential: string): string | null {
  try {
    const params = new URLSearchParams(credential)
    return params.get('Email') || null
  } catch (error) {
    console.error('Failed to parse credential:', error)
    return null
  }
}

export async function refreshCredentials() {
  const config = await ConfigManager.GetConfig()
  credentialMap.value = {}
  accountOptions.value = []

  if (config.credentials?.length) {
    config.credentials.forEach(credential => {
      const email = extractEmailFromCredential(credential)
      if (email) {
        credentialMap.value[email] = credential
        accountOptions.value.push(email)
      }
    })
  }

  if (config.selected) {
    selectedAccount.value = config.selected
    await updateTokenBindingPrompt(config.selected)
  }
}

async function updateTokenBindingPrompt(email: string) {
  const credential = credentialMap.value[email]
  if (!credential) {
    tokenBindingEmail.value = ''
    return
  }
  tokenBindingEmail.value = await ConfigManager.CredentialNeedsTokenBinding(credential) ? email : ''
}

export async function selectAccount(email: string) {
  await ConfigManager.SetSelected(email)
  await updateTokenBindingPrompt(email)
}

export async function addCredentials(authString: string): Promise<boolean> {
  try {
    await ConfigManager.AddCredentials(authString)

    const email = extractEmailFromCredential(authString)
    if (email) {
      credentialMap.value[email] = authString
      if (!accountOptions.value.includes(email)) {
        accountOptions.value = [...accountOptions.value, email]
      }
      selectedAccount.value = email
      await updateTokenBindingPrompt(email)
    }
    toast.success('Credentials added successfully!')
    return true
  } catch (error: unknown) {
    reportError(error, 'Failed to add credentials')
    return false
  }
}

export async function addGoogleAccount(oauthToken: string): Promise<string> {
    const email = await ConfigManager.AddGoogleAccount(oauthToken)
    await refreshCredentials()
    return email
}

export async function removeCredentials(email: string): Promise<boolean> {
  try {
    await ConfigManager.RemoveCredentials(email)

    if (credentialMap.value[email]) {
      delete credentialMap.value[email]
      accountOptions.value = accountOptions.value.filter(opt => opt !== email)
      if (selectedAccount.value === email) {
        selectedAccount.value = ''
      }
      if (tokenBindingEmail.value === email) {
        tokenBindingEmail.value = ''
      }
    }
    toast.success('Credentials removed.')
    return true
  } catch (error) {
    reportError(error, 'Failed to remove credentials')
    return false
  }
}

export async function addTokenBindingAliasFromADB() {
  if (!tokenBindingEmail.value) return

  isExtractingTokenBinding.value = true
  try {
    await ConfigManager.AddTokenBindingAliasFromADB(tokenBindingEmail.value)
    await refreshCredentials()
    tokenBindingEmail.value = ''
    toast.success('Token binding key added.')
  } catch (error) {
    reportError(error, 'Failed to add token binding key')
  } finally {
    isExtractingTokenBinding.value = false
  }
}

export function useCredentials() {
  return {
    selectedAccount,
    accountOptions,
    tokenBindingEmail,
    isExtractingTokenBinding,
    refreshCredentials,
    selectAccount,
    addCredentials,
    addGoogleAccount,
    removeCredentials,
    addTokenBindingAliasFromADB,
  }
}

Events.On('accountChanged', () => {
  void refreshCredentials()
})
