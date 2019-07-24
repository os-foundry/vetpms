import 'babel-polyfill';
import Vue from "vue";

Vue.config.productionTip = false;
Vue.config.devtools = true;

// Setup Vuetify
import Vuetify from 'vuetify';
Vue.use(Vuetify);

import 'vuetify/dist/vuetify.min.css';
import 'material-design-icons-iconfont';

// Setup router
import Router from 'vue-router';
Vue.use(Router);
import Home from '@/components/Home';
import UserCreate from '@/components/UserCreatePage';

const router = new Router({
    routes: [
        { path: '/', name: 'home', component: Home },
        { path: '/users/new', name: 'newUser', component: UserCreate },
    ]
});

// Setup i18n
import VueI18n from 'vue-i18n';
Vue.use(VueI18n);
import { messages } from './translations';

const i18n = new VueI18n({
    locale: 'en',
    messages
});

// Connect App
import App from './App.vue';
import Bridge from './wailsbridge';

Bridge.Start(() => {
    new Vue({
        router,
        i18n,
        render: h => h(App)
  }).$mount("#app");
});
