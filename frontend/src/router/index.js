import Vue from 'vue'
import Router from 'vue-router'

Vue.use(Router)

import Layout from '@/views/Layout'

export const constantRouterMap = [{
    path: '',
    component: Layout,
    redirect: 'containers',
    children: [{
      path: 'containers',
      component: () => import('@/views/Containers'),
      name: 'containers',
      meta: {
        title: 'containers',
      }
    }]
  },
  {
    path: '/terminals',
    component: Layout,
    children: [{
      path: '',
      component: () => import('@/views/Terminals'),
      name: 'terminals',
      meta: {
        title: 'terminals',
      }
    }]
  },
  {
    path: '/test',
    component: Layout,
    children: [{
      path: '',
      component: () => import('@/views/Test'),
      name: 'test',
      meta: {
        title: 'test',
      }
    }]
  }
]

export default new Router({
  mode: 'history',
  routes: constantRouterMap
})
