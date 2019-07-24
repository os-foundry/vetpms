<template>
  <div>
  <v-layout justify-center
            align-center
            class="px-0"
            v-for="(message, i) in messages"
            :key="i"
  >
    <v-alert :type="message.type"
             :value="message.text"
             border="left"
             colored-border
             elevation="2"
             class="wide"
    >
      {{ message.text }}
    </v-alert>
  </v-layout>
  </div>
</template>

<script>
  import { EventBus } from '../eventbus.js';

  export default {
    data: () => ({
      messages: [],
    }),
    mounted () {
      window.wails.Events.On("ALERT", message => {
        this.show(message);
      });

      EventBus.$on("ALERT", message => {
        this.show(message);
      });
    },
    methods: {
      show(message) {
        this.messages.push(message);
        setTimeout(() => {
          this.messages.shift();
        }, message.timeout && message.timeout != 0 ? message.timeout : 3000);
      }
    }
  };
</script>

<style scoped>
  .wide {
    min-width: 50%;
  }
</style>
