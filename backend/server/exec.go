package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"backend/utils"
	log "github.com/sirupsen/logrus"
)

func ExecServerRun(port string) {
	http.HandleFunc("/api/resize", execResize)
	http.HandleFunc("/ws", execWS)
	log.Info("start exec server...")
	log.Fatal(http.ListenAndServe(net.JoinHostPort("", port), nil))
}

func execWS(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cookie := r.URL.Query()["cookie"][0]
	termID := r.URL.Query()["termID"][0]
	containerID := r.URL.Query()["containerID"][0]
	user := r.URL.Query()["user"][0]
	cols, _ := strconv.Atoi(r.URL.Query()["cols"][0])
	rows, _ := strconv.Atoi(r.URL.Query()["rows"][0])

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(fmt.Sprintf("upgrade error: %v", err))
		return
	}
	defer c.Close()

	err = utils.ExecCon(c, containerID, termID, cookie, user, cols, rows)
	if err != nil {
		log.Error(err)
	}
}

func execResize(w http.ResponseWriter, r *http.Request) {
	cookie := r.URL.Query()["cookie"][0]
	termID := r.URL.Query()["termID"][0]
	cols, _ := strconv.Atoi(r.URL.Query()["cols"][0])
	rows, _ := strconv.Atoi(r.URL.Query()["rows"][0])
	utils.ExecResize(termID, cookie, cols, rows)

	w.Write([]byte("resize..."))
}
