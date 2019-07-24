<template>
  <app-page title="Dashboard">
    <v-tabs
          v-model="model"
          color="primary"
          slider-color="yellow"
          centered
        >
          <v-tab
            v-for="i in 3"
            :key="i"
            :href="`#tab-${i}`"
          >
            <v-icon>phone</v-icon>&nbsp;
            Item {{ i }}
          </v-tab>
    </v-tabs>
    <v-tabs-items v-model="model">
      <v-tab-item
        v-for="i in 3"
        :key="i"
        :value="`tab-${i}`"
      >
        <v-card flat>
          <v-card-text v-text="text"></v-card-text>
        </v-card>
      </v-tab-item>
    </v-tabs-items>
  </app-page>
</template>

<script>
  import AppPage from "@/components/Page";

  export default {
    components: {AppPage},
    data () {
      return {
        model: 'tab-2',
        text: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.'
      }
    }
  }
</script>

<style scoped>
  #page {
    max-width: 75%;
    height: 80%;
  }
</style>
