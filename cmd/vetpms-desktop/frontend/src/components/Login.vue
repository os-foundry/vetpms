<template>
  <v-container>
    <v-layout justify-center>
      <v-dialog v-model="showLogin" persistent width="500">
        <v-toolbar color="primary">
          <v-toolbar-title>
            {{ $t("login.title") }}
          </v-toolbar-title>
        </v-toolbar>
        <v-card>
          <v-container>
            <v-form v-model="valid" ref="form" lazy-validation @keyup.native.enter="login">
              <v-text-field
                :label='$t("general.email")'
                v-model="userName"
                required
                :rules="emailRules"
              ></v-text-field>

              <v-text-field
                :label='$t("general.password")'
                v-model="password"
                type="password"
                required
                :rules="passwordRules"
              ></v-text-field>

              <v-checkbox
                :label='$t("login.remember")'
                v-model="rememberMe">
              </v-checkbox>

              <v-btn
                @click="login"
                :disabled="!valid || loading"
              >
                {{ $t("login.signin") }}
              </v-btn>

              <v-btn
                flat
                @click="reset"
                :disabled="loading"
              >
                {{ $t("general.resetform") }}
              </v-btn>

            </v-form>
          </v-container>
        </v-card>
      </v-dialog>
    </v-layout>
  </v-container>
</template>

<script>
  export default {
      mounted() {
        window.backend.Core.CurrentUser().then(user => {
            this.user = user;
        });

        window.wails.Events.On("LOGIN", (user) => {
            this.user = user;
            this.reset();
        });

        window.wails.Events.On("LOGOUT", () => {
            this.user = null;
            this.reset();
        });
        this.mountDone = true;
      },
      computed: {
        hasLogin() {
            return this.user != null;
        },
        showLogin() {
            return !this.hasLogin && this.mountDone;
        },
      },
      data () { return {
        loading: false,
        user: null,
        mountDone: false,
        valid: true,
        userName: "",
        password: "",
        rememberMe: false,
        emailRules: [
            v => !!v || this.$t("validation.required", {field: this.$t("general.email")}),
            v => /.+@.+/.test(v) || this.$t("validation.valid",
                                    {field: this.$t("general.email")})
        ],
        passwordRules: [
            v => !!v || this.$t("validation.valid", {field: this.$t("general.password")})
        ],
      }},
      methods: {
        login() {
            if (this.$refs.form.validate()) {
              this.loading = false;
              window.backend.Core.Login(this.userName, this.password).then(() => {
                  this.loading = false;
                  this.reset();
              }).catch(e => {
                  this.loading = false;
              });
            }
       },
       reset() {
         this.error = "";
         this.$refs.form.reset()
       },
    }
  }
</script>
