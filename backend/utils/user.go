package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type UserResp struct {
	Data struct {
		Code   string `json:"code"`
		Depart struct {
			Name string `json:"name"`
		} `json:"depart"`
		Name string `json:"name"`
	} `json:"data"`
}

type UserInfo struct {
	Code       string `json:"code"`
	DepartName string `json:"departname"`
	Name       string `json:"name"`
}

var userMap = NewSafeMap()

func GetUserByCode(code string) *UserInfo {
	u := userMap.Get(code)
	if u != nil {
		return u.(*UserInfo)
	}

	user := &UserInfo{Code: code}
	req, err := http.NewRequest("GET", "http://tyrande-gateway.nidianwo.com/tyrande/api/staffs/"+code, nil)
	if err != nil {
		log.Error(err)
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("failed to get user info from http://tyrande-gateway.nidianwo.com/tyrande/api/staffs/%s, err:%v", code, err))
	}
	defer resp.Body.Close()
	ret := new(UserResp)
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}
	err = json.Unmarshal(data, ret)
	if err != nil {
		log.Error(fmt.Sprintf("failed to unmarshal data: %s", string(data)))
	} else {
		user.Name = ret.Data.Name
		user.DepartName = ret.Data.Depart.Name
		userMap.Set(code, user)
	}

	return user
}
