<template>
  <div class="layout">
    <div class="layout-header">
      <el-menu :default-active="activeIndex" class="hmenu" mode="horizontal" @select="handleSelect" background-color="#545c64"
        text-color="#fff" active-text-color="#ffd04b">
        <el-menu-item index="containers">容器列表</el-menu-item>
        <el-menu-item index="terminals">终端</el-menu-item>
        <!-- <el-tooltip :content="userInfo" effect="dark" placement="bottom" style="float:right;margin-right: 8px;margin-top: 13px;"> -->
        <el-tooltip effect="dark" placement="bottom" style="float:right;margin-right: 8px;margin-top: 13px;">
          <div slot="content" v-html=userInfo></div>
          <a href="/api/logout">
            <svg class="icon" aria-hidden="true" style="font-size: 30px;" color="#ffd04b">
              <use xlink:href="#l-icon-people"></use>
            </svg>
          </a>
        </el-tooltip>
      </el-menu>
    </div>
    <div class="main-container">
      <keep-alive>
        <router-view />
      </keep-alive>
    </div>
  </div>
</template>

<script>
  export default {
    data() {
      return {
        activeIndex: this.$route.matched[1].name,
        userInfo: ""
      }
    },
    created() {
      this.getUser();
    },
    methods: {
      handleSelect(key, keyPath) {
        this.$router.push({
          name: key
        })
      },
      getUser() {
        var self = this;
        this.$ajax({
            method: 'get',
            url: '/api/user',
          })
          .then(function (response) {
            // console.log(response.data)
            let user = response.data
            self.userInfo = "姓名:" + user.name + "<br/>" + "工号:" + user.code;
          })
          .catch(function (response) {
            console.log(response)
          });
      }
    },
    watch: {
      $route: function () {
        this.$nextTick(() => {
          this.activeIndex = this.$route.matched[1].name
        })
      }
    }
  }

</script>

<style scoped>
  .layout {
    /* position: relative; */
    height: auto;
    min-height: 100%;
    /* overflow: auto; */
    /* height: 100%; */
    /* background-color: rgb(240, 242, 245); */
  }

  .main-container {
    /* position: relative; */
    /* height: 10%; */
    min-height: calc(100vh - 93px);
    padding: 16px;
    background-color: rgb(240, 242, 245);
  }

</style>
