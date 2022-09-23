package utils

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/fsouza/go-dockerclient"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type clientSession struct {
	client *docker.Client
	execID string
}

func ExecCon(ws *websocket.Conn, containerID, termID, cookie, user string, cols, rows int) error {
	writeLock := &sync.Mutex{}

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		SendCloseMessage(ws, ERROR_CONN, writeLock)
		return err
	}

	shellType, err := getShell(client, containerID)
	if err != nil {
		SendCloseMessage(ws, ERROR_CONN, writeLock)
		return err
	}

	execCmd := []string{"env", "TERM=xterm-256color", shellType}
	opts := docker.CreateExecOptions{
		// Container: "d2bde51dd7ae",
		Container:    containerID,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		User:         user,
		Cmd:          execCmd,
	}

	var exec *docker.Exec
	if exec, err = client.CreateExec(opts); err != nil {
		return err
	}

	stdinPipeReader, stdinPipeWriter := io.Pipe()
	stdoutPipeReader, stdoutPipeWriter := io.Pipe()
	stderrPipeReader, stderrPipeWriter := io.Pipe()

	wg := &sync.WaitGroup{}
	wg.Add(3)
	go handleIn(ws, stdinPipeWriter, wg, termID, cookie)
	go handleOut(ws, stdoutPipeReader, wg, writeLock, termID, cookie)
	go handleOut(ws, stderrPipeReader, wg, writeLock, termID, cookie)

	termSet(termID, cookie, &clientSession{
		client: client,
		execID: exec.ID,
	})
	successCh := make(chan struct{})
	go firstResize(successCh, termID, cookie, cols, rows)
	if err := client.StartExec(exec.ID, docker.StartExecOptions{
		Detach:       false,
		OutputStream: stdoutPipeWriter,
		ErrorStream:  stderrPipeWriter,
		InputStream:  stdinPipeReader,
		RawTerminal:  false,
		Success:      successCh,
	}); err != nil {
		log.Error(fmt.Sprintf("failed to start docker exec, %v", err))
		SendCloseMessage(ws, ERROR_CONN, writeLock)
	}

	log.Info("docker exec finished...")

	stdoutPipeWriter.Close()
	stderrPipeWriter.Close()
	stdinPipeReader.Close()
	ws.Close()
	wg.Wait()
	return nil
}

func firstResize(successCh chan struct{}, termID, cookie string, cols, rows int) {
	<-successCh
	ExecResize(termID, cookie, cols, rows)
	close(successCh)
}

func ExecResize(termID, cookie string, cols, rows int) {
	if termMap.Check(cookie) && termMap.Get(cookie).(*SafeMap).Check(termID) {
		session := termMap.Get(cookie).(*SafeMap).Get(termID).(*clientSession)
		err := session.client.ResizeExecTTY(session.execID, rows, cols)
		if err != nil {
			log.Error(fmt.Sprintf("failed to get exec session with cookie: %s, id: %s, err: %v", cookie, termID, err))
		}
		log.Info(fmt.Sprintf("resize exec tty size, cookie: %s, id: %s, cols: %d, rows: %d", cookie, termID, cols, rows))
	} else {
		log.Error(fmt.Sprintf("failed to get exec session with cookie: %s, id: %s", cookie, termID))
	}
}

func getShell(client *docker.Client, containerID string) (string, error) {
	execCmd := []string{"cat", "/etc/shells"}
	opts := docker.CreateExecOptions{
		// Container: "d2bde51dd7ae",
		Container:    containerID,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          execCmd,
	}

	var exec *docker.Exec
	var err error
	if exec, err = client.CreateExec(opts); err != nil {
		return "", err
	}
	var stdout bytes.Buffer
	if err := client.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: &stdout,
	}); err != nil {
		return "", err
	}

	shellType := "/bin/sh"
	if strings.Contains(stdout.String(), "/bin/bash") {
		shellType = "/bin/bash"
	}
	return shellType, nil
}
