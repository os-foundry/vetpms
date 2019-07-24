<template>
  <div>
    <v-navigation-drawer v-model="drawer" clipped fixed app>
      <v-toolbar flat class="transparent" v-if="hasLogin">
        <v-list class="pa-0">
          <v-list-tile avatar>
            <v-list-tile-avatar>
              <img src="https://randomuser.me/api/portraits/lego/2.jpg">
            </v-list-tile-avatar>

            <v-list-tile-content>
              <v-list-tile-title>{{ user.name }}</v-list-tile-title>
            </v-list-tile-content>
          </v-list-tile>
        </v-list>
      </v-toolbar>

      <v-list dense>

        <v-list-tile to="/">
          <v-list-tile-action>
            <v-icon>dashboard</v-icon>
          </v-list-tile-action>
          <v-list-tile-content>
            <v-list-tile-title>{{ $t("menu.dashboard") }}</v-list-tile-title>
          </v-list-tile-content>
        </v-list-tile>

        <v-list-group
          v-for="(menu, i) in menus"
          :key="i"
          :prepend-icon="menu.icon"
        >
          <template v-slot:activator>
            <v-list-tile>
              <v-list-tile-title>{{ $t(menu.title) }}</v-list-tile-title>
            </v-list-tile>
          </template>

          <v-list-tile
            v-for="(item, ii) in menu.items"
            :key="ii"
            :to="item.to"
            @click=""
            v-if="userHasRole(item.roles)"
          >
          <v-list-tile-title v-text="$t(item.title)"></v-list-tile-title>
            <v-list-tile-action>
              <v-icon v-text="item.icon"></v-icon>
            </v-list-tile-action>
          </v-list-tile>
        </v-list-group>

        <v-list-tile>
          <v-list-tile-action>
            <v-icon>settings</v-icon>
          </v-list-tile-action>
          <v-list-tile-content>
            <v-list-tile-title>{{ $t("menu.settings") }}</v-list-tile-title>
          </v-list-tile-content>
        </v-list-tile>

      </v-list>
    </v-navigation-drawer>
    <v-toolbar app fixed clipped-left>
      <v-toolbar-side-icon @click.stop="drawer = !drawer"></v-toolbar-side-icon>
      <v-toolbar-title>{{ toolbarTitle }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn flat @click="logout" v-if="hasLogin">{{ $t("login.signout") }}</v-btn>
    </v-toolbar>

    <app-login></app-login>
  </div>
</template>

<script>
  import AppLogin from "@/components/Login";

  export default {
    components: { AppLogin },
    computed: {
        hasLogin() {
            return this.user != null;
        },
    },
    mounted() {
        window.backend.Core.CurrentUser().then(user => {
            this.user = user;
        });

        window.wails.Events.On("LOGIN", (user) => {
          this.user = user;
        });

        window.wails.Events.On("LOGOUT", () => {
          this.user = null;
        });
    },
    props: {
      toolbarTitle: String,
    },
    data () { return {
        user: null,
        loading: false,
        drawer: false,
        menus: [
            {
                title: "menu.users",
                icon: "account_circle",
                items: [
                    { title: "menu.usersManagement", icon: "people_outline", roles: ["ADMIN"], to: "/users" },
                    { title: "menu.usersNew", icon:"add", roles: ["ADMIN"], to: "/users/new"  }
                ]
            }
        ],
    }},
    methods: {
        userHasRole(roles) {
            // Return false if there is no logged in user
            if(!this.hasLogin) { return false; }

            var res = false;
            roles.forEach(g => {
                this.user.roles.forEach(r => {
                    if(r==g) { res = true; return; }
                });
                if(res == true) { return; }
            });
            return res;
        },
        logout() {
            this.loading = true;
            window.backend.Core.Logout();
            this.loading = false;
        }
    }
};
</script>
