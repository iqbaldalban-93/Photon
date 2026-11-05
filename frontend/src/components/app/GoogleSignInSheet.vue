<script setup lang="ts">
import { ref, watch } from 'vue'
import { Browser } from '@wailsio/runtime'
import { DialogDescription, DialogTitle } from 'reka-ui'
import { toast } from 'vue-sonner'
import { ExternalLink } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { describe } from '../../utils/errors'
import { useCredentials } from '../../stores/credentials'

const embeddedSetupURL = 'https://accounts.google.com/EmbeddedSetup'

const credentials = useCredentials()
const isOpen = defineModel<boolean>('open', { default: false })

const oauthToken = ref('')
const isConnecting = ref(false)

watch(isOpen, (open) => {
  if (!open) {
    oauthToken.value = ''
  }
})

async function openEmbeddedSetup() {
  try {
    await Browser.OpenURL(embeddedSetupURL)
  } catch (error) {
    toast.error('Could not open the system browser', { description: describe(error) })
  }
}

async function connectAccount() {
  const token = oauthToken.value.trim()
  if (!token || isConnecting.value) return

  isConnecting.value = true
  try {
    const email = await credentials.addGoogleAccount(token)
    isOpen.value = false
    toast.success('Google account connected', { description: email })
  } catch (error) {
    toast.error('Could not connect Google account', { description: describe(error) })
  } finally {
    oauthToken.value = ''
    isConnecting.value = false
  }
}
</script>

<template>
  <Sheet v-model:open="isOpen">
    <SheetContent
      side="bottom"
      class="max-h-[92vh] overflow-y-auto"
      style="--wails-draggable: none"
    >
      <div class="mx-auto flex w-full max-w-sm flex-col px-5 pt-5 pb-6">
        <DialogTitle class="font-display pb-1 text-center text-lg font-semibold">
          Sign in to Google Photos
        </DialogTitle>
        <DialogDescription class="pb-4 text-center text-xs text-muted-foreground">
          Embedded setup — no mobile app capture needed.
        </DialogDescription>

        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-2">
            <p class="flex items-center gap-2 text-sm font-medium select-none">
              <span class="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted text-xs">1</span>
              Open the Google sign-in page
            </p>
            <Button
              type="button"
              variant="outline"
              class="cursor-pointer select-none"
              :disabled="isConnecting"
              @click="openEmbeddedSetup"
            >
              <ExternalLink class="size-4" />
              Open Google sign-in
            </Button>
            <ol class="list-decimal space-y-1 rounded-lg border bg-muted/30 py-2.5 pr-3 pl-7 text-xs leading-snug text-muted-foreground">
              <li>
                Sign in with the account you want to add, then click
                <span class="text-foreground">I agree</span>. The page may keep loading forever; that's expected.
              </li>
              <li>Open the browser developer tools, then Application or Storage → Cookies → <code class="text-foreground">accounts.google.com</code>.</li>
              <li>Copy only the value of the <code class="text-foreground">oauth_token</code> cookie.</li>
            </ol>
          </div>

          <div class="flex flex-col gap-2">
            <Label
              for="google-oauth-token"
              class="flex items-center gap-2 text-sm font-medium"
            >
              <span class="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted text-xs">2</span>
              Paste the oauth_token value
            </Label>
            <Input
              id="google-oauth-token"
              v-model="oauthToken"
              type="password"
              autocomplete="off"
              spellcheck="false"
              placeholder="Paste the cookie value"
              :disabled="isConnecting"
              @keydown.enter="connectAccount"
            />
            <Button
              type="button"
              class="cursor-pointer select-none"
              :disabled="!oauthToken.trim() || isConnecting"
              @click="connectAccount"
            >
              {{ isConnecting ? 'Connecting…' : 'Connect account' }}
            </Button>
          </div>
        </div>
      </div>
    </SheetContent>
  </Sheet>
</template>
