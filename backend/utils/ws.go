package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var wsPort = "2222"

func WSCon(ws *websocket.Conn, hostAddr, containerID, termID, cookie, user string, cols, rows int) error {
	u := url.URL{Scheme: "ws", Host: hostAddr + ":" + wsPort, Path: "/ws", RawQuery: "cookie=" + cookie + "&termID=" + termID + "&containerID=" + containerID + "&user=" + user + "&cols=" + strconv.Itoa(cols) + "&rows=" + strconv.Itoa(rows)}
	// u := url.URL{Scheme: "ws", Host: "localhost" + ":" + wsPort, Path: "/ws", RawQuery: "cookie=" + cookie + "&termID=" + termID + "&containerID=" + containerID + "&user=" + user + "&cols=" + strconv.Itoa(cols) + "&rows=" + strconv.Itoa(rows)}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()

	wg := &sync.WaitGroup{}
	// handle wesocket in message
	wg.Add(1)
	go handleWSMsg(ws, c, wg)

	// handle websocket out message
	wg.Add(1)
	go handleWSMsg(c, ws, wg)

	termSet(termID, cookie, hostAddr)
	wg.Wait()
	termDel(termID, cookie)
	return nil
}

func handleWSMsg(wsIn, wsOut *websocket.Conn, wg *sync.WaitGroup) {
	defer wsIn.Close()
	defer wsOut.Close()
	for {
		_, message, err := wsIn.ReadMessage()
		if err != nil {
			log.Error(fmt.Sprintf("read from websocket error: %v", err))
			break
		}
		err = wsOut.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			log.Error(fmt.Sprintf("write to websocket error: %v", err))
			break
		}
	}
	wg.Done()
}

func RemoteResize(termID, cookie string, cols, rows int) {
	remoteAddr := ""
	if termMap.Check(cookie) && termMap.Get(cookie).(*SafeMap).Check(termID) {
		remoteAddr = termMap.Get(cookie).(*SafeMap).Get(termID).(string)
	} else {
		log.Error(fmt.Sprintf("failed to get remote addr with cookie: %s, id: %s", cookie, termID))
	}
	req, _ := http.NewRequest("GET", "http://"+remoteAddr+":"+wsPort+"/api/resize?"+"cookie="+cookie+"&termID="+termID+"&cols="+strconv.Itoa(cols)+"&rows="+strconv.Itoa(rows), nil)
	// req, _ := http.NewRequest("GET", "http://"+"localhost"+":"+wsPort+"/api/resize?"+"cookie="+cookie+"&termID="+termID+"&cols="+strconv.Itoa(cols)+"&rows="+strconv.Itoa(rows), nil)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	_, err := client.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("failed to request remote server to resize term, remote addr: %s, err: %v", remoteAddr, err))
	}
}
