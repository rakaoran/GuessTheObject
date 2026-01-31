<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { endpoints } from '../config'
import PublicGamesList from '../components/PublicGamesList.vue'

const router = useRouter()
const isLoggedIn = ref(false)
const username = ref('')

onMounted(() => {
  const storedUser = localStorage.getItem('username')
  if (storedUser) {
    isLoggedIn.value = true
    username.value = storedUser
  }
})

const navigateTo = (path: string) => {
  router.push(path)
}

const handleLogout = async () => {
  try {
    await fetch(endpoints.logout, { method: 'POST', credentials: 'include' })
  } catch (err) {
    console.error("Logout error:", err)
  } finally {
    localStorage.removeItem('username')
    isLoggedIn.value = false
    username.value = ''
    router.push('/')
  }
}
</script>

<template>
  <div class="min-h-screen bg-[#0F1115] text-[#E6E6E6] font-sans flex flex-col">

    <header class="container mx-auto px-4 py-8 md:py-12 flex flex-col items-center text-center max-w-6xl">

      <div
        class="mb-6 p-4 bg-[#171A21] rounded-2xl border border-[#242833] shadow-sm transform hover:scale-105 transition-transform duration-300">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round" class="w-12 h-12 text-[#4C8DFF]">
          <path d="M12 19l7-7 3 3-7 7-3-3z"></path>
          <path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z"></path>
          <path d="M2 2l7.586 7.586"></path>
          <circle cx="11" cy="11" r="2"></circle>
        </svg>
      </div>

      <h1 class="text-4xl md:text-6xl font-black mb-4 tracking-tighter text-[#E6E6E6]">
        GUESS THE <span class="text-transparent bg-clip-text bg-gradient-to-r from-[#4C8DFF] to-[#2ecc71]">OBJECT</span>
      </h1>

      <p class="text-lg md:text-xl text-[#A0A4AB] mb-10 max-w-2xl leading-relaxed">
        Draw, Guess, Win! The ultimate real-time multiplayer drawing showdown.
      </p>

      <div class="w-full max-w-5xl mx-auto">
        <template v-if="isLoggedIn">
          <div class="grid grid-cols-1 md:grid-cols-12 gap-6 w-full text-left">

            <!-- Left Column: Actions -->
            <div class="md:col-span-4 flex flex-col gap-4">
              <!-- Create Game Button -->
              <button @click="navigateTo('/create')"
                class="relative overflow-hidden group w-full bg-[#171A21] border border-[#242833] hover:border-[#4C8DFF]/50 p-6 rounded-2xl shadow-xl transition-all duration-300 text-left">
                <div
                  class="absolute inset-0 bg-gradient-to-br from-[#4C8DFF]/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity">
                </div>
                <div class="relative z-10 flex flex-col items-start">
                  <div class="p-3 bg-[#1F232D] rounded-xl mb-3 text-[#4C8DFF]">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8" viewBox="0 0 24 24" fill="none"
                      stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                      <line x1="12" y1="8" x2="12" y2="16"></line>
                      <line x1="8" y1="12" x2="16" y2="12"></line>
                    </svg>
                  </div>
                  <h2 class="text-xl font-bold text-white mb-1">Create Game</h2>
                  <p class="text-sm text-[#A0A4AB]">Start a new match with custom rules.</p>
                </div>
              </button>

              <!-- Join by Code Button -->
              <button @click="navigateTo('/join')"
                class="relative overflow-hidden group w-full bg-[#171A21] border border-[#242833] hover:border-[#2ecc71]/50 p-6 rounded-2xl shadow-xl transition-all duration-300 text-left">
                <div
                  class="absolute inset-0 bg-gradient-to-br from-[#2ecc71]/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity">
                </div>
                <div class="relative z-10 flex flex-col items-start">
                  <div class="p-3 bg-[#1F232D] rounded-xl mb-3 text-[#2ecc71]">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8" viewBox="0 0 24 24" fill="none"
                      stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path>
                      <polyline points="10 17 15 12 10 7"></polyline>
                      <line x1="15" y1="12" x2="3" y2="12"></line>
                    </svg>
                  </div>
                  <h2 class="text-xl font-bold text-white mb-1">Join by Code</h2>
                  <p class="text-sm text-[#A0A4AB]">Have a room code? Enter it here.</p>
                </div>
              </button>

              <div class="mt-2 text-center">
                <button @click="handleLogout"
                  class="text-sm text-[#A0A4AB] hover:text-[#E6E6E6] underline decoration-dotted underline-offset-4 transition-colors">
                  Sign out as {{ username }}
                </button>
              </div>
            </div>

            <!-- Right Column: Public Games List -->
            <div class="md:col-span-8 h-full min-h-[500px]">
              <PublicGamesList />
            </div>
          </div>
        </template>

        <template v-else>
          <div class="bg-[#171A21]/50 backdrop-blur-sm border border-[#242833] rounded-2xl p-8 max-w-lg mx-auto">
            <div class="flex flex-col sm:flex-row gap-4 w-full justify-center">
              <button @click="navigateTo('/signup')"
                class="px-8 py-4 bg-[#4C8DFF] hover:bg-[#3b7cdb] text-white font-bold rounded-xl transition-all shadow-lg hover:shadow-[#4C8DFF]/25 transform hover:-translate-y-0.5">
                Sign Up
              </button>
              <button @click="navigateTo('/login')"
                class="px-8 py-4 bg-[#121419] border border-[#242833] hover:border-[#4C8DFF] text-[#E6E6E6] hover:text-[#4C8DFF] font-bold rounded-xl transition-all hover:bg-[#171A21]">
                Login
              </button>
            </div>
            <p class="mt-6 text-sm text-[#A0A4AB]">
              Join thousands of players painting their way to victory!
            </p>
          </div>
        </template>

      </div>
    </header>

    <!-- Features Section - Only show when not logged in or at the very bottom -->
    <section v-if="!isLoggedIn" class="container mx-auto px-6 py-12 max-w-6xl grow">
      <div class="grid md:grid-cols-3 gap-6">
        <div
          class="bg-[#171A21] p-8 border border-[#242833] rounded-xl hover:border-[#4C8DFF]/50 transition-colors duration-300 group">
          <div
            class="mb-5 flex items-center justify-center w-12 h-12 bg-[#242833] rounded-lg text-[#4C8DFF] group-hover:scale-110 transition-transform">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 19l7-7 3 3-7 7-3-3z"></path>
              <path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z"></path>
              <path d="M2 2l7.586 7.586"></path>
            </svg>
          </div>
          <h3 class="text-lg font-bold mb-2 text-[#E6E6E6]">Draw & Express</h3>
          <p class="text-[#A0A4AB] text-sm leading-relaxed">Choose a word and bring it to life using our smooth,
            responsive
            canvas tools. No artistic skills required—just imagination.</p>
        </div>

        <div
          class="bg-[#171A21] p-8 border border-[#242833] rounded-xl hover:border-[#2ecc71]/50 transition-colors duration-300 group">
          <div
            class="mb-5 flex items-center justify-center w-12 h-12 bg-[#242833] rounded-lg text-[#2ecc71] group-hover:scale-110 transition-transform">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"></circle>
              <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
              <line x1="12" y1="17" x2="12.01" y2="17"></line>
            </svg>
          </div>
          <h3 class="text-lg font-bold mb-2 text-[#E6E6E6]">Guess Fast</h3>
          <p class="text-[#A0A4AB] text-sm leading-relaxed">Analyze the live stream of drawings and type your answer.
            Speed
            matters—the faster you guess, the more points you earn.</p>
        </div>

        <div
          class="bg-[#171A21] p-8 border border-[#242833] rounded-xl hover:border-[#f39c12]/50 transition-colors duration-300 group">
          <div
            class="mb-5 flex items-center justify-center w-12 h-12 bg-[#242833] rounded-lg text-[#f39c12] group-hover:scale-110 transition-transform">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 20v-6M6 20V10M18 20V4"></path>
            </svg>
          </div>
          <h3 class="text-lg font-bold mb-2 text-[#E6E6E6]">Rank Up</h3>
          <p class="text-[#A0A4AB] text-sm leading-relaxed">Create private rooms for friends or join public matches.
            Dominate the leaderboard and showcase your skills to the world.</p>
        </div>
      </div>
    </section>

    <footer class="text-center py-8 border-t border-[#242833] bg-[#0F1115] mt-auto">
      <p class="text-sm text-[#A0A4AB]">
        © {{ new Date().getFullYear() }} Guess The Object. All rights reserved.
      </p>
    </footer>
  </div>
</template>