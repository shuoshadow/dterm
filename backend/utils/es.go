package utils

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Code     int
	RespData []byte
}

type ESclient struct {
	Addr   string
	Client *http.Client
}

func NewESClient() *ESclient {
	return &ESclient{
		Addr: os.Getenv("ES_ADDR"),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *ESclient) Do(method string, url string, buf *bytes.Buffer) (*Response, error) {
	req, err := http.NewRequest(method, "http://"+e.Addr+"/"+url, buf)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}

	resp, err := e.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	ret := new(Response)
	ret.Code = resp.StatusCode

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		ret.RespData = data
	}

	return ret, err
}

func (e *ESclient) CheckIndex(indexName string) (bool, error) {
	var buf bytes.Buffer
	resp, err := e.Do("HEAD", indexName, &buf)
	if err != nil {
		return false, err
	}
	// fmt.Println(resp.Code, string(resp.RespData))
	if resp.Code == http.StatusNotFound {
		return false, nil
	}
	return true, nil
}

func (e *ESclient) ReIndex(indexName string) error {
	var buf bytes.Buffer
	_, err := e.Do("DELETE", indexName, &buf)
	if err != nil {
		return err
	}
	bufIndex := bytes.NewBuffer([]byte(INDEX_TEMPLATE))
	resp, err := e.Do("PUT", indexName, bufIndex)
	if err != nil {
		return err
	} else if resp.Code != http.StatusOK {
		return errors.New(string(resp.RespData))
	}

	return nil
}

var INDEX_TEMPLATE = `{
  "mappings": {
    "podtype": {
      "properties": {
        "metadata": {
          "properties": {
            "name": {
              "type": "text",
              "fields": {
                "keyword": {
                  "type": "keyword",
                  "ignore_above": 256,
                  "normalizer": "my_normalizer"
                }
              }
            },
            "namespace": {
              "type": "text",
              "fields": {
                "keyword": {
                  "type": "keyword",
                  "ignore_above": 256,
                  "normalizer": "my_normalizer"
                }
              }
            },
            "ownerReferences": {
              "properties": {
                "name": {
                  "type": "text",
                  "fields": {
                    "keyword": {
                      "type": "keyword",
                      "ignore_above": 256,
                      "normalizer": "my_normalizer"
                    }
                  },
                  "fielddata": true
                }
              }
            }
          }
        },
        "status": {
          "properties": {
            "phase": {
              "type": "text",
              "fields": {
                "keyword": {
                  "type": "keyword",
                  "ignore_above": 256,
                  "normalizer": "my_normalizer"
                }
              }
            }
          }
        }
      }
    }
  },
  "settings":{
    "index":{
      "number_of_shards":1,
      "number_of_replicas":0,
      "analysis": {
        "normalizer": {
          "my_normalizer": {
            "filter": [
              "lowercase",
              "asciifolding"
            ],
            "type": "custom",
            "char_filter": [
            ]
          }
        }
      }
    }
  }
}`

var SEARCH_TEMPLATE = `{
  "size":0,
  "query" : {
    "bool": {
      "should": [
        {
          "wildcard": {
            "metadata.name.keyword": {
              "value": "*%s*"
            }
          }
        },{
          "wildcard": {
            "metadata.namespace.keyword": {
              "value": "*%s*"
            }
          }
        },{
          "wildcard": {
            "metadata.ownerReferences.name.keyword": {
              "value": "*%s*"
            }
          }
        },{
          "wildcard": {
            "status.phase.keyword": {
              "value": "*%s*"
            }
          }
        },{
          "wildcard": {
            "status.podIP.keyword": {
              "value": "*%s*"
            }
          }
        },{
          "wildcard": {
            "status.hostIP.keyword": {
              "value": "*%s*"
            }
          }
        }
      ]
    }
  },
  "aggs":{
    "owner_references":{
      "terms":{
		"size":2147483647,
        "field":"metadata.ownerReferences.name.keyword",
        "order": {
          "max_time" : "desc" 
        }
      },
      "aggs":{
        "max_time": {
          "max": {
            "field": "metadata.creationTimestamp"
          } 
        },
        "references":{
          "top_hits":{
            "size": 100,
            "_source": {
              "includes": [
                "metadata.name" ,
                "metadata.namespace" ,
                "status.podIP",
                "status.hostIP",
                "metadata.creationTimestamp",
				"status.phase",
				"status.containerStatuses.name",
                "status.containerStatuses.state",
                "status.containerStatuses.ready",
                "status.containerStatuses.restartCount",
				"metadata.ownerReferences.name",
				"metadata.ownerReferences.kind"]
            },
            "sort":[
              {
                "metadata.creationTimestamp":{
                  "order":"desc"
                }
              }
            ]
          }
        }
      }
    }
  }
}`
