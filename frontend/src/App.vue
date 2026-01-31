<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterView } from 'vue-router';
import { endpoints } from './config';

onMounted(async () => {
  try {
    // Attempt to refresh the session on startup
    const response = await fetch(endpoints.refresh, {
      method: 'GET',
      credentials: 'include', // Important to send the existing httpOnly cookie
    });

    if (response.ok) {
      console.log('Session refreshed successfully');
    } else {
      console.log('No active session or session expired');
      localStorage.removeItem('username');
    }
  } catch (err) {
    console.error('Failed to refresh session:', err);
  }
});
</script>

<template>
  <RouterView />
</template>

<style scoped></style>
