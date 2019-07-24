<template>
  <app-page title="New user">
    <v-container>
      <user-form @user-form-event="save"></user-form>
    </v-container>
  </app-page>
</template>

<script>
  import AppPage from "@/components/Page";
  import UserForm from "@/components/UserForm";

  export default {
    components: {AppPage, UserForm},
    methods: {
      save(user) {
          window.backend.Core.UserCreate(JSON.stringify(user)).then((newUser) => {
              console.log(newUser);
          });
      }
    }
  }
</script>
