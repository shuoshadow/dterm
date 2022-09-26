// The Vue build version to load with the `import` command
// (runtime-only or standalone) has been set in webpack.base.conf with an alias.
import Vue from 'vue'

import Element, {
  Form
} from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import './assets/icon/iconfont'
import './assets/icon/iconfont.css'

import axios from 'axios';
import {
  Terminal
} from 'xterm';
import "xterm/dist/xterm.css";
import * as fit from 'xterm/lib/addons/fit/fit';
import * as search from 'xterm/lib/addons/search/search';
import * as zmodem from 'xterm/lib/addons/zmodem/zmodem';
require('zmodem.js/dist/zmodem');
import * as attach from 'xterm/lib/addons/attach/attach';
import Base64Tool from 'js-base64';

import '@/styles/index.scss'

import App from './App'
import router from './router'
import store from './store'

Vue.config.productionTip = false

Terminal.applyAddon(fit);
Terminal.applyAddon(attach);
Terminal.applyAddon(search);
Terminal.applyAddon(zmodem);
Vue.prototype.$xterm = Terminal;
Vue.prototype.$base64 = Base64Tool;

axios.defaults.baseURL = process.env.BASE_API
Vue.prototype.$ajax = axios;

axios.interceptors.request.use(
  config => {
    let tempRouter = router
    if (tempRouter.currentRoute.query.ticket) {
      config.url = config.url + "?ticket=" + tempRouter.currentRoute.query.ticket;
      config.url = config.url + "&currentRoute=" + tempRouter.currentRoute.fullPath;
    }
    return config;
  },
  err => {
    return Promise.reject(err);
  });

axios.interceptors.response.use(
  response => {
    return response;
  },
  error => {
    if (error.response) {
      switch (error.response.status) {
        case 401:
          window.location.href = "https://xx.xxx.com/login?service=" + process.env.BASE_API + router.currentRoute.path
      }
    }
    return Promise.reject(error.response.data)
  });

Vue.use(Element, {
  size: 'medium'
})

/* eslint-disable no-new */
new Vue({
  el: '#app',
  router,
  store,
  render: h => h(App)
})
