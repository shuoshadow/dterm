<template>
  <div id="term-test1">
    <el-button size="mini" type="primary" @click="newTab" style="margin-bottom:20px">登陆</el-button>
    <el-button size="mini" type="primary" @click="testevent" style="margin-bottom:20px">resize</el-button>
    <!-- <input ref="upload" id="zm_files" type="file" multiple> -->
    <div class="tip">
      <span style="font-size: 12px;color: #5e6d82;">&nbsp;应用: <code>item.deployID</code>
        &nbsp;&nbsp;环境: <code>item.nameSpace</code>
        &nbsp;&nbsp;实例IP: <code>item.ipAddress</code></span>
      <span style="font-size: 12px;color: #5e6d82;margin-left:20px" id="percent_received"></span>
      <input ref="upload" id="zm_files" type="file" multiple style="margin-left:20px;display:none;">
    </div>
    <div id="test111"></div>
  </div>
</template>
<script>
  //   require('zmodem.js/dist/zmodem');
  //   import * as zmodemdev from 'zmodem.js/dist/zmodem';
  export default {
    data() {
      return {
        termObject: {},
        socketObject: {},
        testEvent: false
      };
    },
    computed: {},
    mounted() {
      var self = this;
      window.onresize = function temp() {
        self.termResize();
      };
    },
    watch: {
      testEvent: function () {
        console.log(this.testEvent);
        console.log(this.$refs.upload);
        this.$nextTick(function () {
          this.$refs.upload.dispatchEvent(new MouseEvent("click"));
        });
      }
    },
    methods: {
      handleReceive(zsession) {
        function _save_to_disk(xfer, buffer) {
          return Zmodem.Browser.save_to_disk(buffer, xfer.get_details().name);
        }

        function _update_progress(xfer) {
          var file_info = xfer.get_details();
          var total_in = xfer.get_offset();
          var percent_received = (100 * total_in) / xfer.get_details().size;
          //   console.log(percent_received.toFixed(2))
          let progressContent =
            "下载进度: " +
            percent_received.toFixed(2) +
            "% (" +
            total_in +
            "/" +
            file_info.size +
            ")bytes  " +
            file_info.name;
          document.getElementById(
            "percent_received"
          ).textContent = progressContent;
        }

        let _this = this;
        zsession.on("offer", function (xfer) {
          function on_form_submit() {
            //START
            var FILE_BUFFER = [];
            xfer.on("input", payload => {
              _update_progress(xfer);
              FILE_BUFFER.push(new Uint8Array(payload));
            });
            xfer.accept().then(() => {
              console.log("before save...");
              _save_to_disk(xfer, FILE_BUFFER);
              console.log("after save...");
              // _this.termObject.attach(_this.socketObject);
            }, console.error.bind(console));
            //END
          }

          on_form_submit();
        });

        var promise = new Promise(res => {
          zsession.on("session_end", () => {
            res();
          });
        });

        zsession.start();

        return promise;
      },
      handleSend(zsession) {},
      termResize() {
        let h = String(document.body.clientHeight - 189 - 26) + "px";
        let w = String(document.body.clientWidth - 64) + "px";
        let elTab = document.getElementById("test111");
        elTab.style.height = h;
        elTab.style.width = w;
        this.termObject.fit();
      },
      testevent() {
        // this.$refs.upload.dispatchEvent(new MouseEvent('click'))
        this.testEvent = true;
        // this.$forceUpdate()
        console.log(this.$refs.upload);
      },
      resize(c, r) {
        // var self = this;
        // let h = String(document.body.clientHeight - 189 - 26)
        // let w = String(document.body.clientWidth - 64)
        this.$ajax({
            method: "get",
            url: "/api/resize",
            params: {
              cols: c,
              rows: r
            }
          })
          .then(function (response) {
            console.log(response.data);
          })
          .catch(function (response) {
            console.log(response);
          });
      },
      newTab() {
        // this.$refs.upload.dispatchEvent(new MouseEvent('click'))

        function _save_to_disk(xfer, buffer) {
          return Zmodem.Browser.save_to_disk(buffer, xfer.get_details().name);
        }

        function handle_receive(zsession) {
          zsession.on("offer", function (xfer) {
            function on_form_submit() {
              //START
              var FILE_BUFFER = [];
              xfer.on("input", payload => {
                // console.log("push to buffer...")
                FILE_BUFFER.push(new Uint8Array(payload));
              });
              xfer.accept().then(() => {
                _save_to_disk(xfer, FILE_BUFFER);
                console.log("save to disk...");
                // Zmodem.Browser.save_to_disk(FILE_BUFFER, xfer.get_details().name);
                //   Zmodem.Browser.save_to_disk(FILE_BUFFER, xfer.get_details().name);
              }, console.error.bind(console));
              //END
            }

            on_form_submit();
            console.log("finished...");
          });

          var promise = new Promise(res => {
            zsession.on("session_end", () => {
              res();
            });
          });

          zsession.start();

          return promise;
        }

        function _update_progress(xfer) {
          var file_info = xfer.get_details();
          var total_in = xfer.get_offset();
          var percent_received = (100 * total_in) / xfer.get_details().size;
          //   console.log(percent_received.toFixed(2))
          let progressContent =
            "上传进度: " +
            percent_received.toFixed(2) +
            "% (" +
            total_in +
            "/" +
            file_info.size +
            ")bytes  " +
            file_info.name;
          document.getElementById(
            "percent_received"
          ).textContent = progressContent;
        }

        function handle_send(zsession) {
          var file_el = document.getElementById("zm_files");
          file_el.style.display = "";
          var promise = new Promise(res => {
            file_el.onchange = function (e) {
              var files_obj = file_el.files;

              Zmodem.Browser.send_files(zsession, files_obj, {
                  on_offer_response(obj, xfer) {
                    // if (xfer) _show_progress();
                    //console.log("offer", xfer ? "accepted" : "skipped");
                    console.log("on offer...");
                  },
                  on_progress(obj, xfer) {
                    _update_progress(xfer);
                    console.log("on progress..");
                  },
                  on_file_complete(obj) {
                    //console.log("COMPLETE", obj);
                    // _hide_progress();
                    document.getElementById("zm_files").style.display = "none";
                    console.log("on complete...");
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
        }

        let elTab = document.getElementById("test111");
        // console.log(elTab)
        elTab.style.height = String(document.body.clientHeight - 189 - 26) + "px";

        let socket = new WebSocket("ws://localhost:8082/echo");
        // let socket = new WebSocket("ws://localhost:8083/echo");

        let xterm = new this.$xterm({
          cursorBlink: true
        });
        xterm.open(elTab, true);
        xterm.fit();
        xterm.focus();

        socket.onopen = function () {
          xterm.attach(socket);
        };

        let self = this;

        // zmodem test
        xterm.zmodemAttach(socket, {
          noTerminalWriteOutsideSession: true
        });

        xterm.on("zmodemRetract", () => {
          console.log("retract...");
        });

        xterm.on("zmodemDetect", detection => {
          xterm.detach();
          let zsession = detection.confirm();

          var promise;

          if (zsession.type === "receive") {
            console.log("sz...");
            // promise = handle_receive(zsession);
            promise = this.handleReceive(zsession);
          } else {
            console.log("rz...");
            // xterm.blur()
            // self.$refs.upload.dispatchEvent(new MouseEvent('click'))
            // self.$nextTick(function () {
            //   self.$message.error('获取列表失败！');
            // })
            self.testEvent = true;

            promise = handle_send(zsession);
          }

          promise.catch(console.error.bind(console)).then(() => {
            console.log("reattach...");
            xterm.attach(socket);
          });
        });
        // end

        var termSize = xterm.proposeGeometry();
        console.log(termSize);

        // socket.onmessage = function (message) {
        //   var reader = new FileReader()
        //   reader.addEventListener('loadend', function () {
        //     var dec = new TextDecoder('utf-8')
        //     var cont = JSON.parse(dec.decode(reader.result))
        //     if (cont.msgType === 1) {
        //       xterm.writeln(
        //         '\x1B[31m' +
        //         self.$base64.Base64.decode(cont.content) +
        //         '\x1B[0m'
        //       )
        //       // socket.close();
        //     } else {
        //       xterm.write(self.$base64.Base64.decode(cont.content))
        //     }
        //   })
        //   reader.readAsArrayBuffer(message.data)
        // }
        // socket.onopen = function () {
        //   let msg = {
        //     msgType: 1,
        //     content: self.$base64.Base64.encode(
        //       termSize.cols + ' ' + termSize.rows
        //     )
        //   }
        //   socket.send(JSON.stringify(msg))
        // }
        socket.onclose = function () {
          xterm.writeln(
            "\x1B[31m" + "终端已退出，如需操作请重新登录。" + "\x1B[0m"
          );
        };

        // xterm.on('data', function (data) {
        //   let msg = {
        //     msgType: 0,
        //     content: self.$base64.Base64.encode(data)
        //   }
        //   socket.send(JSON.stringify(msg))
        // })

        xterm.on("resize", function (data) {
          self.resize(data.cols, data.rows);
        });

        this.termObject = xterm;
        this.socketObject = socket;
      }
    },
    created() {
      //   this.newTab()
    }
  };

</script>
<style scoped>
  .term-test {
    padding: 16px;
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
