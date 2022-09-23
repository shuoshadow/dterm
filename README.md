## 项目介绍

---
k8s的容器shell登录平台, 支持多集群pod同步。


## 项目结构

---

```
.
├── Dockerfile-dterm # dterm镜像
├── Dockerfile-exec # exec镜像
├── README.md
├── backend # 后端
│   ├── ENV_DEV
│   ├── dterm.go
│   ├── dterm_exec.go
│   ├── go.mod
│   ├── go.sum
│   ├── server
│   └── utils
├── build-dterm.sh # 构建dterm镜像
├── build-exec.sh # 构建exec镜像
└── frontend # 前端
    ├── README.md
    ├── build
    ├── config
    ├── index.html
    ├── package.json
    └── src

```