package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"backend/utils"
	log "github.com/sirupsen/logrus"
)

type esEvent struct {
	template map[string]map[string]string
	event    interface{}
}

type esEventList []*esEvent

func Sync(config *utils.K8sConfig) {
	k8sClient := utils.NewK8sClient(config)
	esClient := utils.NewESClient()
	ch := k8sClient.WatchPod("")
	sendList := esEventList{}
	chSend := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(time.Second * 5):
				chSend <- true
			}
		}
	}()
	for {
		select {
		case e := <-ch:
			log.Info(fmt.Sprintf("received an event from pod watch channel, type:%s, podname:%s", e.Type, e.Event.Name))
			switch e.Type {
			case "ADDED", "MODIFIED":
				meta := map[string]map[string]string{
					"index": {
						"_index": "pods",
						"_type":  "podtype",
						"_id":    e.Event.Namespace + ":" + e.Event.Name,
					},
				}
				sendList = append(sendList, &esEvent{
					template: meta,
					event:    e.Event,
				})
			case "DELETED":
				meta := map[string]map[string]string{
					"delete": {
						"_index": "pods",
						"_type":  "podtype",
						"_id":    e.Event.Namespace + ":" + e.Event.Name,
					},
				}
				sendList = append(sendList, &esEvent{
					template: meta,
					event:    nil,
				})
			default:
				log.Error(fmt.Sprintf("failed to handle event, type: %v, event: %v", e.Type, e.Event))
			}
		case <-chSend:
			log.Debug(fmt.Sprintf("in time case, slice length is %d", len(sendList)))
			if len(sendList) > 0 {
				//check index 'pods' exist, if not, reindex it
				exists, err := esClient.CheckIndex("pods")
				if err != nil {
					log.Error(fmt.Sprintf("failed to check index exists, %v", err))
				} else if !exists {
					log.Info("index not found, reindex it...")
					err = esClient.ReIndex("pods")
					if err != nil {
						log.Error(fmt.Sprintf("failed to reindex 'pods', %v", err))
					}
				}

				// send bulk request to es
				log.Info(fmt.Sprintf("%d event(s) send to es", len(sendList)))
				var buf bytes.Buffer
				for _, event := range sendList {
					jtemplate, _ := json.Marshal(event.template)
					buf.Write(jtemplate)
					buf.WriteByte('\n')
					if event.event != nil {
						jevent, err := json.Marshal(event.event)
						if err != nil {
							log.Error(err)
							continue
						}
						buf.Write(jevent)
						buf.WriteByte('\n')
						// log.Warn(string(jevent))
					}
				}
				resp, err := esClient.Do("POST", "_bulk", &buf)
				if err != nil {
					log.Error(err)
				} else {
					log.Debug(fmt.Sprintf("es code: %d, data: %v", resp.Code, string(resp.RespData)))
				}
				sendList = esEventList{}
			}
		}
	}
}
