<script setup lang="ts">
const SHOW_BANNER = true
</script>

<template>
  <div v-if="SHOW_BANNER"
    class="w-full bg-[#ff4c4c]/15 border border-[#ff4c4c]/40 rounded-2xl p-6 text-center">
    <h2 class="text-2xl md:text-4xl font-black text-[#ff4c4c] mb-2">
      SERVER IS CURRENTLY DOWN
    </h2>
    <p class="text-base md:text-lg text-[#A0A4AB]">
      Keeping it hosted costs money lol. Check back later!
    </p>
  </div>
</template>
