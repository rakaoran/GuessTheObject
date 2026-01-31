<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { endpoints } from '../config'

const router = useRouter()
const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const errorMsg = ref('')
const isLoading = ref(false)

const handleSignUp = async () => {
  errorMsg.value = ''

  if (password.value !== confirmPassword.value) {
    errorMsg.value = "Passwords don't match!"
    return
  }
  if (password.value.length < 8) {
    errorMsg.value = "Password must be at least 8 characters long."
    return
  }
  const usernameRegex = /^[a-z0-9_]{3,20}$/i;
  if (!usernameRegex.test(username.value)) {
    errorMsg.value = "Username must be 3-20 characters (letters, numbers, underscores only)."
    return
  }

  isLoading.value = true

  try {
    const response = await fetch(endpoints.signup, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({
        username: username.value,
        password: password.value
      })
    })

    const data = await response.text()
    const errorMessage = data || 'Signup failed'

    if (!response.ok) {
      if (errorMessage === 'username-already-exists') {
        throw new Error('This username is already taken.')
      }
      if (errorMessage === 'weak-password') {
        throw new Error('Password is too weak.')
      }
      if (errorMessage === 'password-too-long') {
        throw new Error('Password is too long.')
      }
      if (errorMessage === 'invalid-username-format') {
        throw new Error('Invalid username format.')
      }
      if (errorMessage === 'server-timeout') {
        throw new Error('Server is taking too long to respond')
      }
      throw new Error(errorMessage)
    }

    localStorage.setItem('username', username.value)
    router.push('/')

  } catch (err: any) {
    errorMsg.value = err.message
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div
    class="min-h-screen flex items-center justify-center bg-[#0F1115] p-4 text-[#E6E6E6] font-sans selection:bg-[#4C8DFF] selection:text-white">

    <div class="w-full max-w-md bg-[#171A21] border border-[#242833] rounded-xl shadow-2xl overflow-hidden">

      <div class="px-8 pt-8 pb-2">
        <h2 class="text-2xl font-bold text-[#E6E6E6]">
          Create an account
        </h2>
        <p class="text-[#A0A4AB] text-sm mt-2">
          Join to start drawing and guessing.
        </p>
      </div>

      <form @submit.prevent="handleSignUp" class="p-8 space-y-5">

        <div v-if="errorMsg"
          class="p-3 bg-red-500/10 border border-red-500/20 text-red-400 text-sm rounded flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <span>{{ errorMsg }}</span>
        </div>

        <div class="space-y-1.5">
          <label class="block text-[#E6E6E6] text-sm font-medium" for="username">
            Username
          </label>
          <input v-model="username"
            class="w-full bg-[#0F1115] text-[#E6E6E6] px-4 py-2.5 border border-[#242833] rounded-lg focus:border-[#4C8DFF] outline-none placeholder-[#A0A4AB] transition-colors"
            id="username" type="text" placeholder="Choose a username" required>
          <p class="text-xs text-[#A0A4AB]">
            3-20 characters, letters, numbers & underscores only.
          </p>
        </div>

        <div class="space-y-1.5">
          <label class="block text-[#E6E6E6] text-sm font-medium" for="password">
            Password
          </label>
          <input v-model="password"
            class="w-full bg-[#0F1115] text-[#E6E6E6] px-4 py-2.5 border border-[#242833] rounded-lg focus:border-[#4C8DFF] outline-none placeholder-[#A0A4AB] transition-colors"
            id="password" type="password" placeholder="Min. 8 characters" required>
        </div>

        <div class="space-y-1.5">
          <label class="block text-[#E6E6E6] text-sm font-medium" for="confirmPassword">
            Confirm Password
          </label>
          <input v-model="confirmPassword"
            class="w-full bg-[#0F1115] text-[#E6E6E6] px-4 py-2.5 border border-[#242833] rounded-lg focus:border-[#4C8DFF] outline-none placeholder-[#A0A4AB] transition-colors"
            id="confirmPassword" type="password" placeholder="Re-enter password" required>
        </div>

        <button
          class="w-full bg-[#4C8DFF] hover:bg-[#3b7cdb] text-white font-semibold py-2.5 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed mt-2 shadow-sm"
          type="submit" :disabled="isLoading">
          {{ isLoading ? 'Creating account...' : 'Sign Up' }}
        </button>
      </form>

      <div class="px-8 py-6 border-t border-[#242833] bg-[#121419] flex justify-between items-center text-sm">
        <router-link to="/" class="text-[#A0A4AB] hover:text-[#E6E6E6] transition-colors flex items-center gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M19 12H5"></path>
            <path d="M12 19l-7-7 7-7"></path>
          </svg>
          Cancel
        </router-link>

        <p class="text-[#A0A4AB]">
          Already have an account?
          <router-link to="/login" class="text-[#4C8DFF] hover:underline font-medium ml-1">
            Login
          </router-link>
        </p>
      </div>
    </div>
  </div>
</template>