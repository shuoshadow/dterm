import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

const state = {
  newTerminal: null
}
const getters = {
  getNewTerminal: state => {
    return state.newTerminal;
  }
}
const mutations = {
  setNewTerminal(state, term) {
    state.newTerminal = null;
    state.newTerminal = term;
  }

}
const store = new Vuex.Store({
  state,
  getters,
  mutations
})

export default store
