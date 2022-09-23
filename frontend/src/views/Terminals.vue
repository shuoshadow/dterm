<template>
  <div class="container-tabs">
    <el-tabs v-model="activeTab" type="card" closable @tab-remove="removeTab" @tab-click="clickTab">
      <el-tab-pane v-for="item in tabList" :key="item.name" :label="item.title" :name="item.name" v-if="item.inuse">
        <div class="tip" v-if="item.inuse" :id="'div_tip'+item.title">
          <span style="font-size: 12px;color: #5e6d82;">&nbsp;应用: <code>{{item.deployID}}</code>
            &nbsp;&nbsp;环境: <code>{{item.nameSpace}}</code>
            &nbsp;&nbsp;实例IP: <code>{{item.ipAddress}}</code>&nbsp;&nbsp;&nbsp;内容搜索:</span>
          <el-input :id="'search_input'+item.title" placeholder="请输入内容" size="mini" style="width: 100px;"></el-input>
          <el-button type="primary" icon="el-icon-caret-top" circle @click="handlePreSearch(item.title)" size="mini"
            style="padding:5px;"></el-button>
          <el-button type="primary" icon="el-icon-caret-bottom" circle @click="handleNextSearch(item.title)" size="mini"
            style="margin-left:1px;padding:5px;"></el-button>
          <span style="font-size: 12px;color: #5e6d82;margin-left:20px" :id="'percent_received'+item.title"></span>
          <input ref="upload" :id="'zm_files'+item.title" type="file" multiple style="margin-left:20px;display:none;">
          <el-button size="mini" type="text" circle @click="handleRefresh(item)" style="float:right;padding:0px;"><i
              class="icon iconfont l-icon-refresh" style="font-size: 28px;"></i></el-button>
        </div>
        <div :id="item.name" v-if="item.inuse">
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>
<script>
  import {
    mapGetters
  } from "vuex";
  export default {
    data() {
      return {
        activeTab: "",
        tabList: [],
        termObjects: {}
      };
    },
    computed: {
      ...mapGetters(["getNewTerminal"])
    },
    mounted() {
      let self = this;
      window.onresize = function temp() {
        self.termResize();
      };
    },
    methods: {
      handlePreSearch(label) {
        let searchInput = document.getElementById("search_input" + label);
        this.termObjects[label]["term"].findPrevious(searchInput.value, {
          wholeWord: false,
          caseSensitive: false,
          regex: false
        });
      },
      handleNextSearch(label) {
        let searchInput = document.getElementById("search_input" + label);
        this.termObjects[label]["term"].findNext(searchInput.value, {
          wholeWord: false,
          caseSensitive: false,
          regex: false
        });
      },
      handleRefresh(item) {
        this.termObjects[item.name]["websocket"].close();
        this.termObjects[item.name]["term"].destroy();
        this.newTerm(item);
      },
      termResize() {
        let w = String(document.body.clientWidth - 64) + "px";
        let elTab = document.getElementById(this.activeTab);
        let divTip = document.getElementById("div_tip" + this.activeTab);
        elTab.style.height =
          String(
            document.body.clientHeight -
            61 -
            32 -
            16 -
            41 -
            15 -
            divTip.offsetHeight -
            8
          ) + "px";
        elTab.style.width = w;
        this.termObjects[this.activeTab]["term"].fit();
      },
      resize(c, r, label) {
        this.$ajax({
            method: "get",
            url: "/api/resize",
            params: {
              cols: c,
              rows: r,
              termID: label
            }
          })
          .then(function (response) {
            console.log(response.data);
          })
          .catch(function (response) {
            console.log(response);
          });
      },
      getPod(p) {
        return this.$ajax({
            method: "get",
            url: "/api/pod",
            params: {
              podId: p
            }
          })
          .catch(function (err) {
            console.log(err);
          })
      },
      removeTab(targetName) {
        this.termObjects[targetName]["websocket"].close();
        this.termObjects[targetName]["term"].destroy();
        let tabs = this.tabList.filter(tab => tab.inuse !== false);
        for (let i = 0; i < this.tabList.length; i++) {
          if (this.tabList[i].name === targetName) {
            this.tabList[i]["inuse"] = false;
          }
        }
        let activeName = this.activeTab;
        if (activeName === targetName) {
          tabs.forEach((tab, index) => {
            if (tab.name === targetName) {
              let nextTab = tabs[index + 1] || tabs[index - 1];
              if (nextTab) {
                activeName = nextTab.name;
              }
            }
          });
        }
        this.activeTab = activeName;
        // this.tabList = tabs.filter(tab => tab.name !== targetName);
      },
      clickTab(target) {
        this.$nextTick(() => {
          this.termResize();
          this.termObjects[target.name]["term"].fit();
          this.termObjects[target.name]["term"].focus();
        });
      },
      newTerm(tab) {
        let _this = this;
        this.$ajax({
            method: "get",
            url: "/api/auth"
          })
          .then(function (response) {
            let cookie = response.data;

            let elTab = document.getElementById(tab.name);
            let divTip = document.getElementById("div_tip" + tab.name);
            elTab.style.height =
              String(
                document.body.clientHeight -
                61 -
                32 -
                16 -
                41 -
                15 -
                divTip.offsetHeight -
                8
              ) + "px";

            let xterm = new _this.$xterm({
              cursorBlink: true
            });
            xterm.open(elTab, true);
            xterm.fit();
            xterm.focus();
            let termSize = xterm.proposeGeometry();

            let socket = new WebSocket(
              process.env.BASE_WS_API +
              "/ws?podID=" +
              tab.nameSpace + ":" + tab.podID +
              "&ipAddress=" +
              tab.ipAddress +
              "&cookie=" +
              cookie +
              "&termID=" +
              tab.randomID +
              "&cols=" +
              termSize.cols +
              "&rows=" +
              termSize.rows
            );

            socket.onopen = function () {
              xterm.attach(socket);
            };
            xterm.zmodemAttach(socket, {
              noTerminalWriteOutsideSession: true
            });

            xterm.on("zmodemRetract", () => {
              // console.log("retract...")
            });

            xterm.on("zmodemDetect", detection => {
              xterm.detach();
              let zsession = detection.confirm();

              let promise;

              if (zsession.type === "receive") {
                promise = _this.handleReceive(zsession, tab.name);
              } else {
                _this.$message({
                  showClose: true,
                  message: "点击终端上方按钮选择文件"
                });
                promise = _this.handleSend(zsession, tab.name);
              }

              promise.catch(console.error.bind(console)).then(() => {
                xterm.attach(socket);
              });
            });
            socket.onclose = function () {
              xterm.writeln(
                "\x1B[31m" + "终端已退出，如需操作请重新登录。" + "\x1B[0m"
              );
            };
            xterm.on("resize", function (data) {
              _this.resize(data.cols, data.rows, tab.randomID);
            });

            let termobj = {
              term: xterm,
              websocket: socket
            };
            _this.termObjects[tab.name] = termobj;
          })
          .catch(function (response) {
            console.log(response);
          });
      },
      newTab(term) {
        let count = 0;
        let label = term.podID;
        for (let i = 0; i < this.tabList.length; i++) {
          if (label === this.tabList[i].name.replace(/\[\d+\]$/, "")) {
            count += 1;
          }
        }
        if (count > 0) {
          label = label + "[" + count + "]";
        }
        let tab = {
          title: label,
          name: label,
          randomID: Math.random().toString(36).substr(2),
          inuse: true,
          podID: term.podID,
          deployID: term.deployID,
          ipAddress: term.ipAddress,
          nameSpace: term.nameSpace
        };
        this.tabList.push(tab);
        this.activeTab = tab.name;

        this.$nextTick(() => {
          this.newTerm(tab);
        });
      },
      handleReceive(zsession, label) {
        function _save_to_disk(xfer, buffer) {
          return Zmodem.Browser.save_to_disk(buffer, xfer.get_details().name);
        }

        let _this = this;

        zsession.on("offer", function (xfer) {
          function on_form_submit() {
            let FILE_BUFFER = [];
            xfer.on("input", payload => {
              _this.updateProgress(xfer, "下载", label);
              FILE_BUFFER.push(new Uint8Array(payload));
            });
            xfer.accept().then(() => {
              _save_to_disk(xfer, FILE_BUFFER);
            }, console.error.bind(console));
          }

          on_form_submit();
        });

        let promise = new Promise(res => {
          zsession.on("session_end", () => {
            res();
          });
        });

        zsession.start();

        return promise;
      },
      handleSend(zsession, label) {
        let file_el = document.getElementById("zm_files" + label);
        file_el.style.display = "";
        let _this = this;
        let promise = new Promise(res => {
          file_el.onchange = function (e) {
            let files_obj = file_el.files;
            let self = _this;

            Zmodem.Browser.send_files(zsession, files_obj, {
                on_offer_response(obj, xfer) {
                  // console.log("on offer...")
                },
                on_progress(obj, xfer) {
                  self.updateProgress(xfer, "上传", label);
                },
                on_file_complete(obj) {
                  document.getElementById("zm_files" + label).style.display =
                    "none";
                }
              })
              .then()
              .then(zsession.close.bind(zsession), console.error.bind(console))
              .then(() => {
                res();
              });
          };
        });

        return promise;
      },
      updateProgress(xfer, info, label) {
        let file_info = xfer.get_details();
        let total_in = xfer.get_offset();
        let percent_received = (100 * total_in) / xfer.get_details().size;
        let progressContent =
          info +
          "进度: " +
          percent_received.toFixed(2) +
          "% (" +
          total_in +
          "/" +
          file_info.size +
          ")bytes  " +
          file_info.name;
        document.getElementById(
          "percent_received" + label
        ).textContent = progressContent;
      }
    },
    watch: {
      getNewTerminal(term) {
        this.newTab(term);
      }
    },
    created() {
      let term = this.$store.getters.getNewTerminal;
      if (term !== null) {
        this.newTab(term);
      }

      // for url access
      if (this.$route.query.podId !== undefined) {
        let _this = this
        this.getPod(this.$route.query.podId).then(function (response) {
          if (response.data.found) {
            _this.newTab({
              'deployID': response.data._source.metadata.labels.app,
              'nameSpace': response.data._source.metadata.namespace,
              'podID': response.data._source.metadata.name,
              'ipAddress': response.data._source.status.podIP
            })
          } else if (response.data.found == false) {
            _this.$message.error(_this.$route.query.podId + ' 不存在！')
          } else {
            _this.$message.error('获取信息失败！')
            console.error(response.data)
          }
        })
      }
    }
  };

</script>
<style scoped>
  .container-tabs {
    padding-top: 16px;
    padding-left: 16px;
    padding-right: 16px;
    background: #fff;
    overflow: auto;
    border-radius: 8px;
    min-height: calc(100vh - 109px);
  }

  .tip {
    background: #ecf8ff;
    border-radius: 4px;
    border-left-width: 5px;
    border-left-style: solid;
    border-left-color: rgb(80, 191, 255);
    padding: 4px;
    /* padding-top: 0.5px; */
    /* padding-left: 8px; */
    /* margin-top: 20px; */
    margin-bottom: 8px;
    overflow: auto;
  }

  .tip code {
    background-color: hsla(0, 0%, 100%, 0.7);
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
