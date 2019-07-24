<template>
  <v-layout justify-center>
    <v-flex xs12 sm6>
      <v-toolbar :color="__color" dark>
        <!-- <v-toolbar-side-icon></v-toolbar-side-icon> -->
        <v-toolbar-title>{{ title }}</v-toolbar-title>
        <v-spacer></v-spacer>
        <!--<v-btn icon>
          <v-icon>search</v-icon>
        </v-btn>-->
      </v-toolbar>

      <v-card>
        <slot></slot>
      </v-card>
    </v-flex>
  </v-layout>
</template>

<script>
  export default {
  name: "AppPage",
  computed: {
  __color() {
    return this.color ? this.color : "indigo";
  }
  },
  props: {
    title: String,
    color: String,
  }
}
</script>
