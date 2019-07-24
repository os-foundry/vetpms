<template>
  <v-app id="inspire" dark>
    <app-navigation></app-navigation>
    <v-content>
      <v-container fluid class="px-0">
        <app-alert></app-alert>
        <v-layout justify-center align-center class="px-0" v-if="user != null">
          <router-view></router-view>
        </v-layout>
      </v-container>
    </v-content>

    <v-footer app fixed>
      <v-layout justify-center>
        Copyright &copy; 2019, UAB "Sonemas"
      </v-layout>
    </v-footer>
  </v-app>
</template>

<script>
import AppAlert from "@/components/Alert";
import AppNavigation from "@/components/Navigation";

export default {
    components: {AppAlert, AppNavigation},
    mounted() {
        window.backend.Core.CurrentUser().then(user => {
            this.user = user;
        });

        window.backend.Core.Language().then(lang => {
            this.$i18n.locale = lang;
        });

        window.wails.Events.On("LOGIN", (user) => {
            this.user = user;
        });

        window.wails.Events.On("LOGOUT", () => {
            this.user = null;
        });
    },
    data () { return {
        user: null,
    }},
    methods: {
        userHasRole(roles) {
            // Return false if there is no logged in user
            if(this.user == null) { return false; }

            var res = false;
            roles.forEach(g => {
                this.user.roles.forEach(r => {
                    if(r==g) { res = true; return; }
                });
                if(res == true) { return; }
            });
            return res;
        },
    }
};
</script>
