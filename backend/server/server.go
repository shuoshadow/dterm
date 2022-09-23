package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"backend/utils"
	log "github.com/sirupsen/logrus"
	cas "gopkg.in/cas.v2"
)

var (
	esClient       *utils.ESclient
	staticfs       = http.FileServer(http.Dir("./statics"))
	accessTemplate = "http access user:%s, remote address:%s, path:%s"
)

func Run(port string) {
	esClient = utils.NewESClient()
	// without cas
	// http.HandleFunc("/api/pods", getPods)
	// http.HandleFunc("/ws", ws)
	// log.Info("start server...")
	// log.Fatal(http.ListenAndServe(net.JoinHostPort("", port), nil))

	// with cas
	casUrl := "https://cas.nidianwo.com"
	// casUrl := "http://cas-qa3-in.dwbops.com"
	m := http.NewServeMux()
	m.HandleFunc("/", static)
	m.HandleFunc("/ws", ws)
	m.HandleFunc("/api/resize", resize)
	m.HandleFunc("/api/auth", auth)
	m.HandleFunc("/api/pod", getPod)
	m.HandleFunc("/api/pods", getPods)
	m.HandleFunc("/api/user", getUser)
	m.HandleFunc("/api/logout", logOut)
	url, _ := url.Parse(casUrl)
	client := cas.NewClient(&cas.Options{
		URL:         url,
		SendService: true,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: client.Handle(m),
	}

	log.Info("start server...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func static(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}
	// set no cache
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	staticfs.ServeHTTP(w, r)
}

type SearchResult struct {
	Aggregations struct {
		OwnerReferences struct {
			Buckets []interface{} `json:"buckets"`
		} `json:"owner_references"`
	} `json:"aggregations"`
}

type PodResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Count   int         `json:"count"`
	Data    interface{} `json:"data"`
}

func getPod(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}

	if k, ok := r.URL.Query()["podId"]; !ok || len(k) == 0 {
		w.Write([]byte("e.g. /api/pod?podId=xxx"))
		return
	}
	podId := r.URL.Query()["podId"][0]
	var buf bytes.Buffer
	resp, err := esClient.Do("GET", "pods/podtype/"+podId+"?pretty", &buf)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get pod %s, %v", podId, err)
		log.Error(errMsg)
		w.Write([]byte(errMsg))
	} else {
		w.Write(resp.RespData)
	}
}

func getPods(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}
	// get request params
	page, _ := strconv.Atoi(r.URL.Query()["page"][0])
	size, _ := strconv.Atoi(r.URL.Query()["size"][0])
	searchItem := r.URL.Query()["searchItem"][0]
	// log.Debug(page, size, searchItem)

	// generate response object
	httpResp := &PodResponse{
		Status: "success",
	}

	// generate es search template
	itemList := []interface{}{}
	for index := 0; index < 6; index++ {
		itemList = append(itemList, searchItem)
	}
	template := fmt.Sprintf(utils.SEARCH_TEMPLATE, itemList...)

	// send es request
	resultData := SearchResult{}
	buf := bytes.NewBuffer([]byte(template))
	resp, err := esClient.Do("POST", "pods/podtype/_search", buf)
	if err != nil {
		httpResp.Status = "failed"
		httpResp.Message = fmt.Sprintf("failed to search es, error message: %v", err)
		log.Error(fmt.Sprintf("failed search es, error message: %v", err))
	} else {
		err = json.Unmarshal(resp.RespData, &resultData)
		if err != nil {
			log.Error(fmt.Sprintf("failed to unmarshal es result: %s", string(resp.RespData)))
			httpResp.Status = "failed"
			httpResp.Message = "failed to unmarshal es result"
		} else {
			rList := resultData.Aggregations.OwnerReferences.Buckets
			httpResp.Count = len(rList)

			min := (page - 1) * size
			max := len(rList)
			if page*size > len(rList) || len(rList) < size {
				max = len(rList)
			} else {
				max = page * size
			}
			if min < len(rList) || min == 0 {
				httpResp.Data = rList[min:max]
			} else {
				httpResp.Data = []interface{}{}
				log.Warn("the min size is bigger than the length of result list, return nothing")
			}
		}
	}

	bytes, err := json.Marshal(httpResp)
	if err != nil {
		log.Error(err)
	}
	w.Write(bytes)
}

func logOut(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	cas.RedirectToLogout(w, r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf(accessTemplate, cas.Username(r), r.RemoteAddr, r.URL.Path))
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}
	u := utils.GetUserByCode(cas.Username(r))
	// w.Write([]byte(cas.Username(r)))
	bytes, err := json.Marshal(u)
	if err != nil {
		log.Error(err)
	}
	w.Write(bytes)
}
