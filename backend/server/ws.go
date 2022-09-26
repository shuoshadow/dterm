package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"backend/utils"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	cas "gopkg.in/cas.v2"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type PodIPResponse struct {
	Found  bool `json:"found"`
	Source struct {
		Status struct {
			PodIP             string            `json:"podIP"`
			HostIP            string            `json:"hostIP"`
			ContainerStatuses []ContainerStatus `json:"containerStatuses"`
		} `json:"status"`
	} `json:"_source"`
}

type ContainerStatus struct {
	Name        string `json:"name"`
	ContainerID string `json:"containerID"`
}

var smap = utils.NewSafeMap()

// auth for websocket
func auth(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}
	c, _ := r.Cookie("_cas_session")
	sessionID := c.Value

	smap.Set(sessionID, cas.Username(r))

	w.Write([]byte(sessionID))
}

// handle websocket
func ws(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["cookie"]) > 0 && smap.Check(r.Form["cookie"][0]) {
		code := smap.Get(r.Form["cookie"][0]).(string)
		smap.Delete(r.Form["cookie"][0])

		ipAddress := ""
		hostAddress := ""
		containerID := ""
		containerList := []ContainerStatus{}
		// ipAddress := r.URL.Query()["ipAddress"][0]
		podID := r.URL.Query()["podID"][0]
		termID := r.URL.Query()["termID"][0]
		cols, _ := strconv.Atoi(r.URL.Query()["cols"][0])
		rows, _ := strconv.Atoi(r.URL.Query()["rows"][0])

		if podID == "" {
			log.Error("failed to get podID from the websocket request")
			return
		} else {
			// get ip and containerid from es by podID
			var buf bytes.Buffer
			resp, err := esClient.Do("GET", "pods/podtype/"+podID+"?_source=status.podIP,status.hostIP,status.containerStatuses.name,status.containerStatuses.containerID", &buf)
			if err != nil {
				log.Error("get pod info error")
			} else {
				p := PodIPResponse{}
				if err = json.Unmarshal(resp.RespData, &p); err != nil {
					log.Error(fmt.Sprintf("failed to unmarshal es result: %s", string(resp.RespData)))
				} else {
					if p.Found {
						ipAddress = p.Source.Status.PodIP
						containerList = p.Source.Status.ContainerStatuses
						hostAddress = p.Source.Status.HostIP
					}
				}
			}
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error(fmt.Sprintf("upgrade error: %v", err))
			return
		}
		defer c.Close()

		log.Debug(ipAddress) //ipAddress is not used in docker exec

		// use docker exec(remote server) to link to container
		writeLock := &sync.Mutex{}
		//logUser, authed, privileged := checkAuthed(code)
		logUser, authed, privileged := "dev", true, false
		if !authed {
			log.Info(fmt.Sprintf("user: %s is not allowd to login to %s", code, podID))
			utils.SendCloseMessage(c, utils.ERROR_AUTH, writeLock)
			return
		}
		if !privileged {
			containerList = filterContainers(containerList, "filebeat")
		}
		switch len(containerList) {
		case 0:
			log.Error(fmt.Sprintf("there's no container in the es result, podID: %s", podID))
			return
		case 1:
			containerID = strings.TrimPrefix(containerList[0].ContainerID, "docker://")
		default:
			MsgTemplate := "There's more than one container in this pod.\r\n"
			for i, n := range containerList {
				MsgTemplate += strconv.Itoa(i+1) + "." + n.Name + "\r\n"
			}
			ChooseTemplate := "Choose one to login:"
			c.WriteMessage(websocket.BinaryMessage, []byte(MsgTemplate+ChooseTemplate))
			input := []byte{}
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					log.Error(fmt.Sprintf("read from websocket error: %v", err))
					return
				}
				if bytes.Equal(message, []byte{13}) {
					inputNum, err := strconv.Atoi(string(input))
					input = []byte{}
					if err != nil || inputNum > len(containerList) || inputNum == 0 {
						log.Error(err)
						c.WriteMessage(websocket.BinaryMessage, []byte("\r\nWrong choice, "+ChooseTemplate))
						continue
					}
					containerID = strings.TrimPrefix(containerList[inputNum-1].ContainerID, "docker://")
					c.WriteMessage(websocket.BinaryMessage, []byte("\r\n"))
					break
				} else {
					c.WriteMessage(websocket.BinaryMessage, message)
					input = append(input, message...)
				}
			}
		}

		err = utils.WSCon(c, hostAddress, containerID, termID, r.Form["cookie"][0], logUser, cols, rows)
		if err != nil {
			log.Error(err)
		}
	} else {
		cas.RedirectToLogin(w, r)
	}
}

func resize(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}

	c, _ := r.Cookie("_cas_session")
	sessionID := c.Value
	cols, _ := strconv.Atoi(r.URL.Query()["cols"][0])
	rows, _ := strconv.Atoi(r.URL.Query()["rows"][0])
	termID := r.URL.Query()["termID"][0]

	// docker exec resize
	utils.RemoteResize(termID, sessionID, cols, rows)

	w.Write([]byte("resize..."))
}

// need to do more
func checkAuthed(code string) (string, bool, bool) {
	logUser := "dev"
	authed := false
	privileged := false
	user := utils.GetUserByCode(code)

	// check privileged
	if user.DepartName == "基础运维" {
		return "root", true, true
	}

	// check authed, everyone is authed for now
	authed = true

	return logUser, authed, privileged
}

func filterContainers(cList []ContainerStatus, filter string) []ContainerStatus {
	resultList := []ContainerStatus{}
	for _, c := range cList {
		if !strings.Contains(c.Name, filter) {
			resultList = append(resultList, c)
		}
	}
	return resultList
}
