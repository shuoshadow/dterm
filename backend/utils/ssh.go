package utils

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	ERROR_CONN = "ERROR：无法建立连接\r\n"
	ERROR_COMM = "ERROR：连接异常\n\n"
	ERROR_AUTH = "ERROR：无权限登录\n\n"
)

var termMap = NewSafeMap()

// func termSet(termID, cookie string, session *ssh.Session) {
func termSet(termID, cookie string, session interface{}) {
	tempMap := NewSafeMap()
	if termMap.Check(cookie) {
		tempMap = termMap.Get(cookie).(*SafeMap)
	}
	tempMap.Set(termID, session)
	termMap.Set(cookie, tempMap)
}

func termDel(termID, cookie string) {
	if termMap.Check(cookie) && termMap.Get(cookie).(*SafeMap).Check(termID) {
		termMap.Get(cookie).(*SafeMap).Delete(termID)
	}

	if termMap.Check(cookie) && len(termMap.Get(cookie).(*SafeMap).Items()) == 0 {
		termMap.Delete(cookie)
	}
}

func SSHCon(ws *websocket.Conn, addr, termID, cookie string, cols, rows int) error {
	writeLock := &sync.Mutex{}

	// pod
	config := &ssh.ClientConfig{
		// User: "root",
		User: "dev",
		Auth: []ssh.AuthMethod{
			ssh.Password("RocsXiTo"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr+":22", config)
	if err != nil {
		SendCloseMessage(ws, ERROR_CONN, writeLock)
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		SendCloseMessage(ws, ERROR_CONN, writeLock)
		return err
	}
	defer session.Close()

	stdinPipeReader, stdinPipeWriter := io.Pipe()
	stdoutPipeReader, stdoutPipeWriter := io.Pipe()
	stderrPipeReader, stderrPipeWriter := io.Pipe()

	session.Stdout = stdoutPipeWriter
	session.Stderr = stderrPipeWriter
	session.Stdin = stdinPipeReader

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		SendCloseMessage(ws, ERROR_CONN, writeLock)
		return err
	}

	if err := session.Setenv("LANG", "zh_CN.UTF-8"); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		SendCloseMessage(ws, ERROR_CONN, writeLock)
		return err
	}

	wg := &sync.WaitGroup{}
	wg.Add(3)

	go handleIn(ws, stdinPipeWriter, wg, termID, cookie)
	go handleOut(ws, stdoutPipeReader, wg, writeLock, termID, cookie)
	go handleOut(ws, stderrPipeReader, wg, writeLock, termID, cookie)

	termSet(termID, cookie, session)
	log.Debug(fmt.Sprintf("add term session to term map, cookie: %s, id: %s", cookie, termID))

	if err := session.Wait(); err != nil {
		SendCloseMessage(ws, ERROR_COMM, writeLock)
		return err
	}

	stdoutPipeWriter.Close()
	stderrPipeWriter.Close()
	stdinPipeReader.Close()
	wg.Wait()

	return nil
}

func Resize(termID, cookie string, cols, rows int) {
	if termMap.Check(cookie) && termMap.Get(cookie).(*SafeMap).Check(termID) {
		session := termMap.Get(cookie).(*SafeMap).Get(termID).(*ssh.Session)
		session.WindowChange(rows, cols)
		log.Debug(fmt.Sprintf("resize term window size, cookie: %s, id: %s, cols: %d, rows: %d", cookie, termID, cols, rows))
	} else {
		log.Error(fmt.Sprintf("failed to get term with cookie: %s, id: %s", cookie, termID))
	}
}

func handleIn(ws *websocket.Conn, sessionWriter io.WriteCloser, wg *sync.WaitGroup, termID, cookie string) {
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Error(fmt.Sprintf("read from websocket error: %v", err))
			message = []byte("exit\n")
			_, err = sessionWriter.Write(message)
			if err != nil {
				log.Error(err)
			}
			break
		}
		_, err = sessionWriter.Write(message)
		if err != nil {
			// logs.Error(errors.WithMessage(err, "write to pipe"))
			// logs.Info("close websocket")
			ws.Close()
			break
		}
	}

	// logs.Info("handle in finished")
	termDel(termID, cookie)

	sessionWriter.Close()
	wg.Done()
}

func SendCloseMessage(ws *websocket.Conn, content string, writeLock *sync.Mutex) {
	writeLock.Lock()
	ws.WriteMessage(websocket.BinaryMessage, []byte(content))
	writeLock.Unlock()
}

func handleOut(ws *websocket.Conn, sessionReader io.ReadCloser, wg *sync.WaitGroup, writeLock *sync.Mutex, termID, cookie string) {
	var (
		err  error
		size int
	)
	buf := make([]byte, 10240)
	cursor := 0
	szStart := []byte{114, 122, 13, 42, 42, 24, 66, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 13, 138, 17}
	rzStart := []byte{114, 122, 32, 119, 97, 105, 116, 105, 110, 103, 32, 116, 111, 32, 114, 101, 99, 101, 105, 118, 101, 46, 42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 48, 50, 51, 98, 101, 53, 48, 13, 138, 17}
	szEnd := []byte{13, 42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}
	szCancel := []byte{24, 24, 24, 24, 24, 24, 24, 24, 24, 24, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}
	rzEnd := []byte{42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}
	checkUT8 := true
	sendOverMsg := false
	for {
		if size, err = sessionReader.Read(buf[cursor:]); err == nil || (err == io.EOF && size > 0) {
			validLen := len(buf[:cursor+size])
			//log.Error(buf[cursor : cursor+size])
			if bytes.Equal(buf[cursor:cursor+size], szStart) || bytes.Equal(buf[cursor:cursor+size], rzStart) {
				//log.Error("match sz/rz...")
				checkUT8 = false
			} else if bytes.Equal(buf[cursor:cursor+size], szEnd) || bytes.Equal(buf[cursor:cursor+size], rzEnd) {
				//log.Error("march sz/rz end...")
				checkUT8 = true
			} else if bytes.Equal(buf[cursor:cursor+size], szCancel) {
				//log.Error("match cancel...")
				tempbuf := make([]byte, 10240)
				for i := 0; i < cursor; i++ {
					tempbuf[i] = buf[i]
				}
				for i := 0; i < len(szEnd); i++ {
					tempbuf[cursor+i] = szEnd[i]
				}
				buf = tempbuf
				validLen = cursor + len(szEnd)
				size = len(szEnd)
				sendOverMsg = true
				checkUT8 = true
			} else {

				validLen := getValidUT8Length(buf[:cursor+size], checkUT8)
				if validLen == 0 {
					// logs.Warn("no valid utf8: %s", string(buf[:cursor+size]))
					cursor = cursor + size
					continue
				}
			}

			writeLock.Lock()
			ws.WriteMessage(websocket.BinaryMessage, buf[:validLen])
			if sendOverMsg {
				ws.WriteMessage(websocket.BinaryMessage, []byte{79, 79})
				sendOverMsg = false
			}
			// log.Error(fmt.Sprintf("terminal output: %v", buf[:validLen]))
			writeLock.Unlock()
			cursor = size + cursor - validLen
			tempbuf := make([]byte, 10240)
			for i := 0; i < cursor; i++ {
				tempbuf[i] = buf[validLen+i]
			}
			buf = tempbuf

		} else {
			log.Error(fmt.Sprintf("read from pipe error: %v", err))
			break
		}
	}

	// logs.Info("handle out finished")
	termDel(termID, cookie)

	sessionReader.Close()
	wg.Done()
}

func getValidUT8Length(data []byte, checkUT8 bool) int {
	if !checkUT8 {
		return len(data)
	}

	validLen := 0
	for i := len(data) - 1; i >= 0; i-- {
		if utf8.RuneStart(data[i]) {
			validLen = i
			if utf8.Valid(data[i:]) {
				validLen = len(data)
			}
			break
		}
	}
	return validLen
}
