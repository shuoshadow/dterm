<template>
  <div class="containers-list">
    <div style="margin-top: 15px;">
      <el-input placeholder='输入关键字搜索(名称、环境、IP)' v-model="searchContent" class="input-with-select" @change="search">
        <el-button slot="append" icon="el-icon-search" @click="search"></el-button>
      </el-input>
    </div>
    <template v-for="(tableItem,index) in tableList">
      <div class="tip" :key="'tip'+index">
        <span style="font-size: 12px;color: #5e6d82;">应用名: <code>{{tableItem.references.hits.hits[0]._source.metadata.ownerReferences[0].name}}</code>&nbsp;&nbsp;
          环境: <code>{{tableItem.references.hits.hits[0]._source.metadata.namespace}}</code>&nbsp;&nbsp;
          集群: <code>{{tableItem.references.hits.hits[0]._source.metadata.ownerReferences[0].cluster}}</code></span>
      </div>

      <el-table :key="'talbe'+index" :data="tableItem.references.hits.hits" size="small" border fit style="width: 100%;margin-top: 8px">
        <el-table-column label="实例ID">
          <template slot-scope="scope">
            <span style="margin-left: 10px">{{ scope.row._source.metadata.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="实例IP">
          <template slot-scope="scope">
            <span style="margin-left: 10px">{{ scope.row._source.status.podIP }}</span>
          </template>
        </el-table-column>
        <el-table-column label="宿主机IP">
          <template slot-scope="scope">
            <span style="margin-left: 10px">{{ scope.row._source.status.hostIP }}</span>
          </template>
        </el-table-column>
        <el-table-column label="运行状态">
          <template slot-scope="scope">
            <el-tooltip placement="top">
              <div slot="content" v-html=scope.row._source.status.subMessage></div>
              <el-tag size="medium" type="success" :hit="true" v-if="scope.row._source.status.phase === 'Running'">{{
                scope.row._source.status.phase }}</el-tag>
              <el-tag size="medium" type="danger" :hit="true" v-if="scope.row._source.status.phase !== 'Running'">{{
                scope.row._source.status.phase }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="创建时间">
          <template slot-scope="scope">
            <span style="margin-left: 10px">{{ scope.row._source.metadata.creationTimestamp }}</span>
          </template>
        </el-table-column>
        <el-table-column label="终端操作">
          <template slot-scope="scope">
            <el-button size="mini" type="primary" @click="handleLogin(index, scope.row)">登陆</el-button>
            <!-- <el-button size="mini" type="text" circle @click="handleLogin(index, scope.row)"><i class="icon iconfont l-icon-playon"
                style="font-size: 30px;"></i></el-button> -->
          </template>
        </el-table-column>
      </el-table>
    </template>

    <div class="block" style="float:right;margin-top: 32px">
      <el-pagination @size-change="handleSizeChange" @current-change="handleCurrentChange" :current-page="currentPage"
        :page-sizes="[5, 10, 20, 50]" :page-size="size" layout="total, sizes, prev, pager, next, jumper" :total="totalPage">
      </el-pagination>
    </div>
  </div>
</template>
<script>
  let lodash = require('lodash')
  export default {
    data() {
      return {
        searchContent: '',
        currentPage: 1,
        size: 10,
        totalPage: 0,
        tableList: []
      }
    },
    created() {
      this.getPodList();
    },
    watch: {
      'searchContent': lodash.debounce(function () {
        this.currentPage = 1
        this.getPodList()
      }, 200)
    },
    methods: {
      getPodList() {
        var self = this;
        this.$ajax({
            method: 'get',
            url: '/api/pods',
            params: {
              searchItem: this.searchContent,
              page: this.currentPage,
              size: this.size
            },
          })
          .then(function (response) {
            if (response.data.status === "success") {
              self.tableList = response.data.data
              for (let x = 0; x < self.tableList.length; x++) {
                for (let y = 0; y < self.tableList[x].references.hits.hits.length; y++) {
                  let refName = self.tableList[x].references.hits.hits[y]._source.metadata.ownerReferences[0].name
                  let refKind = self.tableList[x].references.hits.hits[y]._source.metadata.ownerReferences[0].kind
                  let refNameList = refName.split(":")
                  if (refNameList.length === 3) {
                    let tempStr = refNameList[2]
                    if (refKind === "ReplicaSet") {
                      tempStr = tempStr.replace(/-([0-9]|[a-z])+$/, "")
                    }
                    self.tableList[x].references.hits.hits[y]._source.metadata.ownerReferences[0].name = tempStr
                    self.tableList[x].references.hits.hits[y]._source.metadata.ownerReferences[0].cluster =
                      refNameList[
                        0]
                  }
                  let tempData = self.tableList[x].references.hits.hits[y]._source.metadata.creationTimestamp
                  self.tableList[x].references.hits.hits[y]._source.metadata.creationTimestamp = self.timeFormat(
                    tempData)

                  let state = "Running"
                  let subMessage = ""
                  let containerStatuses = self.tableList[x].references.hits.hits[y]._source.status.containerStatuses
                  if (containerStatuses !== undefined) {
                    for (let z = 0; z < containerStatuses.length; z++) {
                      subMessage += "容器: " + containerStatuses[z].name + "<br/>"
                      Object.keys(containerStatuses[z].state).forEach(function (key) {
                        subMessage += "状态: " + key + "<br/>"
                        if (key !== "running" && containerStatuses[z].state[key].reason != "") {
                          state = containerStatuses[z].state[key].reason
                        }
                      })
                      if (containerStatuses[z].restartCount > 0) {
                        subMessage += "重启: " + containerStatuses[z].restartCount + "次<br/>"
                      }
                      if (z < containerStatuses.length - 1) {
                        subMessage += "<br/>"
                      }
                    }
                    self.tableList[x].references.hits.hits[y]._source.status.phase = state
                  }
                  self.tableList[x].references.hits.hits[y]._source.status.subMessage = subMessage
                }
              }
              self.totalPage = response.data.count
            } else {
              self.$message.error('获取列表失败！');
              console.log(response.data.message)
            }
          })
          .catch(function (response) {
            self.$message.error('获取列表失败！');
            console.log(response)
          });
      },
      timeFormat(timeStr) {
        let fmt = "yyyy-MM-dd hh:mm:ss"
        let date = new Date(timeStr)
        var o = {
          "M+": date.getMonth() + 1, //月份   
          "d+": date.getDate(), //日   
          "h+": date.getHours(), //小时   
          "m+": date.getMinutes(), //分   
          "s+": date.getSeconds(), //秒   
          "q+": Math.floor((date.getMonth() + 3) / 3), //季度   
          "S": date.getMilliseconds() //毫秒   
        };
        if (/(y+)/.test(fmt))
          fmt = fmt.replace(RegExp.$1, (date.getFullYear() + "").substr(4 - RegExp.$1.length));
        for (var k in o)
          if (new RegExp("(" + k + ")").test(fmt))
            fmt = fmt.replace(RegExp.$1, (RegExp.$1.length == 1) ? (o[k]) : (("00" + o[k]).substr(("" + o[k]).length)));
        return fmt
      },
      handleLogin(index, row) {
        // console.log(index, row);
        let term = {
          'deployID': row._source.metadata.ownerReferences[0].name,
          'nameSpace': row._source.metadata.namespace,
          'podID': row._source.metadata.name,
          'ipAddress': row._source.status.podIP
        }
        this.$router.push({
          name: "terminals"
        })
        this.$store.commit('setNewTerminal', term)
      },
      handleSizeChange(val) {
        this.size = val
        this.getPodList()
      },
      handleCurrentChange(val) {
        this.currentPage = val
        this.getPodList()
      },
      search() {
        this.getPodList()
      }
    }
  }

</script>
<style scoped>
  .containers-list {
    padding: 16px;
    /* background-color: rgb(193, 194, 197); */
    background: #fff;
    overflow: auto;
    border-radius: 8px;
    min-height: calc(100vh - 125px);
  }

  .tip {
    background: #ecf8ff;
    border-radius: 4px;
    border-left-width: 5px;
    border-left-style: solid;
    border-left-color: rgb(80, 191, 255);
    padding: 8px;
    /* padding-top: 0.5px; */
    /* padding-left: 8px; */
    margin-top: 20px;
    margin-bottom: 4px;
    overflow: auto;
  }

  .tip code {
    background-color: hsla(0, 0%, 100%, .7);
    color: #445368;
    padding-left: 4px;
    padding-right: 4px;
    border: #eaeefb;
    border-width: 1px;
    border-style: solid;
    border-radius: 4px;
    font-size: 13px;
  }

</style>
