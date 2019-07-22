<template>
  <v-app id="inspire" dark>
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

        <v-list-tile>
          <v-list-tile-action>
            <v-icon>dashboard</v-icon>
          </v-list-tile-action>
          <v-list-tile-content>
            <v-list-tile-title>Dashboard</v-list-tile-title>
          </v-list-tile-content>
        </v-list-tile>

        <v-list-group
          v-for="(menu, i) in menus"
          :key="i"
          :prepend-icon="menu.icon"
        >
          <template v-slot:activator>
            <v-list-tile>
              <v-list-tile-title>{{ menu.title }}</v-list-tile-title>
            </v-list-tile>
          </template>

          <v-list-tile
            v-for="(item, ii) in menu.items"
            :key="ii"
            @click=""
            v-if="userHasRole(item.roles)"
          >
          <v-list-tile-title v-text="item.title"></v-list-tile-title>
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
            <v-list-tile-title>Settings</v-list-tile-title>
          </v-list-tile-content>
        </v-list-tile>

      </v-list>
    </v-navigation-drawer>
    <v-toolbar app fixed clipped-left>
      <v-toolbar-side-icon @click.stop="drawer = !drawer"></v-toolbar-side-icon>
      <v-toolbar-title>{{ toolbarTitle }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn flat @click="logout" v-if="hasLogin">Logout</v-btn>
    </v-toolbar>
    <v-content>
      <v-container fluid class="px-0">
        <v-layout justify-center align-center class="px-0">
          <!--<hello-world></hello-world>-->
        </v-layout>
      </v-container>
    </v-content>
    <v-container>
      <v-layout justify-center>
      <v-dialog v-model="showLogin" persistent width="500">
        <v-toolbar color="primary">
            <v-toolbar-title>
                Log in
            </v-toolbar-title>
        </v-toolbar>
        <v-card>
            <v-form v-model="valid" ref="form" lazy-validation @keyup.native.enter="login">
                <v-container>
                    <v-alert outline color="error" icon="warning" :value="error">{{ error }}</v-alert>
                    <v-layout column>

                        <v-text-field
                                label="Email"
                                v-model="userName"
                                required
                                :rules="emailRules"
                        ></v-text-field>

                        <v-text-field
                                label="Password"
                                v-model="password"
                                type="password"
                                required
                                :rules="passwordRules"
                        ></v-text-field>

                        <v-checkbox label="Remember ne" v-model="rememberMe"></v-checkbox>

                    </v-layout>
                </v-container>
            </v-form>
            <v-card-actions>
                <v-flex offset-xs2 mb-2>
                  <v-btn color="primary" @click="login" :disabled="!valid || loading">Sign In</v-btn>
                  <v-btn flat @click="reset" :disabled="loading">Reset</v-btn>
                </v-flex>
            </v-card-actions>
        </v-card>
      </v-dialog>
    </v-layout>
    </v-container>
    <v-footer app fixed>
      <span style="margin-left:1em text-center">Copyright &copy; 2019, UAB "Sonemas"</span>
    </v-footer>
  </v-app>
</template>

<script>

export default {
    computed: {
        hasLogin() {
            return this.user != null;
        },
        showLogin() {
            return !this.hasLogin;
        },
        hasError() {
            return this.error != null;
        }
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
    data: () => ({
        toolbarTitle: "",
        user: null,
        loading: false,
        valid: true,
        userName: "",
        password: "",
        error: null,
        rememberMe: false,
        emailRules: [
            v => !!v || 'E-mail is required',
            v => /.+@.+/.test(v) || 'E-mail must be valid'
        ],
        passwordRules: [
            v => !!v || 'Password is required'
        ],
        drawer: false,
        menus: [
            {
                title: "Users",
                icon: "account_circle",
                items: [
                    { title: "Management", icon: "people_outline", roles: ["ADMIN"] },
                    { title: "New user", icon:"add", roles: ["ADMIN"] },
                    { title: "Update", icon:"update", roles: ["USER", "ADMIN"] }
                ]
            }
        ],
    }),
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
        validate () {
            if (this.$refs.form.validate()) {
                this.snackbar = true;
            }
        },
        reset() {
            this.error = "";
            this.$refs.form.reset()
        },
        login() {
            this.loading = true;
            window.backend.Core.Login(this.userName, this.password).then(() => {
                this.loading = false;
                this.reset();
            }).catch(e => {
                this.loading = false;
                this.error = e;
                setTimeout(() => {
                    this.error = null;
                }, 3000);
            });
        },
        logout() {
            this.loading = true;
            window.backend.Core.Logout().then(() => {
                this.loading = false;
            })

        }
    }
};
</script>
