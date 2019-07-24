<template>
  <v-form
    autocomplete="false"
    ref="form"
    v-model="valid"
    lazy-validation
  >
    <v-text-field
      v-model="nameFld"
      :counter="100"
      :rules="nameRules"
      label="Name"
      required
    ></v-text-field>

    <v-text-field
      v-model="emailFld"
      :rules="emailRules"
      label="E-mail"
      required
    ></v-text-field>

    <v-text-field
      type="password"
      v-model="passwdFld"
      :rules="passwdRules"
      label="Password"
      required
    ></v-text-field>

    <v-text-field
      type="password"
      v-model="confirmFld"
      :rules="confirmRules"
      label="Confirm password"
      required
    ></v-text-field>

    <!--<v-checkbox
      v-model="adminFld"
      label="Appoint as administrator"
    ></v-checkbox>-->

    <v-btn
      :disabled="!valid"
      @click="validate"
    >
      Save
    </v-btn>

    <v-btn
      flat
      @click="reset"
    >
      Reset Form
    </v-btn>

  </v-form>
</template>

<script>
  export default {
    props: { name: String, email: String, admin: Boolean },

    data () {
      return {
        valid: true,
        nameFld: this.name,
        nameRules: [
          v => !!v || 'Name is required',
          v => (v && v.length <= 100) || 'Name must be less than 100 characters'
        ],
        emailFld: this.email,
        emailRules: [
          v => !!v || 'E-mail is required',
          v => /.+@.+/.test(v) || 'E-mail must be valid'
        ],
        adminFld: this.admin,
        passwdFld: "",
        passwdRules: [
          v => !!v || 'Password is required',
          v => (v && v.length >= 8) || 'Password must not be less than 8 characters'
        ],
        confirmFld: "",
        confirmRules: [
          v => !!v || 'Confirm password is required',
          () => (this.passwdFld === this.confirmFld) || 'Confirm password must match password'
        ],
      }
    },

    methods: {
      validate () {
        if (this.$refs.form.validate()) {
          this.snackbar = true
          this.$emit("user-form-event", {
             name: this.nameFld,
             email: this.emailFld,
             roles: this.adminFld ? ["ADMIN", "USER"] : ["USER"],
             password: this.passwdFld,
             password_confirm: this.confirmFld
          });
          this.$refs.form.reset()
        }
      },
      reset () {
        this.$refs.form.reset()
      },
    },
  }
</script>
