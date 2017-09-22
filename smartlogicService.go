package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bytes"
	"io/ioutil"

	"github.com/Financial-Times/uuid-utils-go"
	log "github.com/sirupsen/logrus"
	"time"
)

type SmartlogicService struct {
	smartlogicAPIKey  string
	smartlogicAddress string
	conceptsRWAddress string
	conceptsBA        string

	notifyURL string
	notifyKey string
}

// Service defines the functions any read-write application needs to implement
type SmartlogicServicer interface {
	Write(prefLabel string) (conceptUUID string, err error)
}

//SmartlogicService instantiate driver
func NewSmartlogicService(smartlogicAPIKey string, smartlogicAddress string, notifyURL string, notifyKey string, conceptsBA string, conceptsURL string) SmartlogicService {
	return SmartlogicService{
		smartlogicAPIKey:  smartlogicAPIKey,
		smartlogicAddress: smartlogicAddress,
		notifyURL:         notifyURL,
		notifyKey:         notifyKey,
		conceptsRWAddress:conceptsURL,
		conceptsBA: conceptsBA,
	}
}

/*
{
	  "@id": "http://www.ft.com/thing/cc9fecf2-dceb-43fc-8d06-06a58105b8b0",
      "@type": [
		"http://www.ft.com/ontology/Topic"
      ],
      "skos:topConceptOf":{
    	"@id":"http://www.ft.com/thing/ConceptScheme/4fd43cbb-bd66-4825-86b8-f9a28d1bc366"
		},
		"sem:guid": [
        {
          "@value": "cc9fecf2-dceb-43fc-8d06-06a58105b8b0"
        }
      ],
      "skosxl:prefLabel": [
      {
         "@type":[
            "skosxl:Label"
         ],
         "skosxl:literalForm":[
            {
               "@language":"en",
               "@value":"Poodles"
            }
         ]
    }
	]
}
*/

type SmartLogicConcept struct {
	Uri          string      `json:"@id"`
	Types        []string    `json:"@type,omitempty"`
	PrefLabels   []PrefLabel `json:"skosxl:prefLabel,omitempty"`
	TopConceptOf RefConcept  `json:"skos:topConceptOf,omitempty"`
	GUUID        []SemGUUID  `json:"sem:guid"`
}

type RefConcept struct {
	Uri string `json:"@id"`
}

type SemGUUID struct {
	Value string `json:"@value"`
}

type PrefLabel struct {
	LitForm []LiteralForm `json:"skosxl:literalForm,omitempty"`
}

type LiteralForm struct {
	Value string `json:"@value"`
}

func getSmartLogicObject(uuid string, prefLabel string) SmartLogicConcept {
	return SmartLogicConcept{
		Uri: "http://www.ft.com/thing/" + uuid,
		Types: []string{
			"http://www.ft.com/ontology/Topic",
		},
		PrefLabels: []PrefLabel{
			PrefLabel{
				LitForm: []LiteralForm{
					LiteralForm{
						Value: prefLabel,
					},
				},
			},
		},
		GUUID: []SemGUUID{
			SemGUUID{
				uuid,
			},
		},
		TopConceptOf: RefConcept{
			Uri: "http://www.ft.com/thing/ConceptScheme/4fd43cbb-bd66-4825-86b8-f9a28d1bc366",
		},
	}
}

func (s SmartlogicService) Write(prefLabel string) (string, error) {
	// TODO generate it randomly
	uuid := uuidutils.NewV3UUID(prefLabel).String()

	smartlogicConceptToWrite := getSmartLogicObject(uuid, prefLabel)
	log.WithFields(log.Fields{"UUID": uuid, "SmartLogicConcept": smartlogicConceptToWrite}).Infof("Sending concept to smartlogic")
	err := sendToSmartlogic(s.smartlogicAddress, uuid, smartlogicConceptToWrite, s.smartlogicAPIKey)

	if err != nil {
		return "", err
	}

	// Hack while the smartlogic notifier doesn't work - Send directly to the concepts-rw
//	err =sendToConceptsRW(s.conceptsRWAddress, uuid, prefLabel, s.conceptsBA)

	//err = sendNotification(s.notifyURL, s.notifyKey)
	//if err != nil {
	//	return "", err
	//}
	return uuid, nil
}

func sendNotification(url string, apiKey string) error {
	client := http.Client{}
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Set("X-Api-Key", apiKey)
	request.Header.Set("X-Request-Id", "hackothon9876")

	q := request.URL.Query()
	q.Add("modifiedGraphId", "1234")
	q.Add("affectedGraphId", "1234")

	now := time.Now()
	now.Add(-10 * time.Millisecond)
	//lastChange, _ := time.Parse("2006-01-02T15:04:05Z", now.String() )
	q.Add("lastChangeDate", "2017-09-22T10:04:05Z")

	request.URL.RawQuery = q.Encode()

	resp, reqErr := client.Do(request)


	log.Infof("Status Code: %v", reqErr)

	if reqErr != nil || resp.StatusCode > 202 {
		readBody, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		err := errors.New("Request to " + url + " returned status: " + strconv.Itoa(resp.StatusCode) + " Response Body: " + string(readBody))
		log.Error(err)
		return err
	}
	return nil
}

func sendToSmartlogic(url string, conceptUUID string, concept SmartLogicConcept, auth string) error {

	client := http.Client{}
	body, err := json.Marshal(concept)

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(concept)

	log.Infof("Sending Body: %v", b)

	if err != nil {
		return err
	}

	request, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))

	log.Infof("Sending Request: %v", request)

	request.Header.Set("authorization", auth)

	// -H 'content-type: application/ld+json' \
	request.Header.Set("content-type", "application/ld+json")
	request.ContentLength = -1
	resp, reqErr := client.Do(request)

	defer resp.Body.Close()

	readBody, _ := ioutil.ReadAll(resp.Body)

	if reqErr != nil || resp.StatusCode/100 != 2 {
		err := errors.New("Request to " + url + " returned status: " + strconv.Itoa(resp.StatusCode) + "; UUID: " + conceptUUID + " Response Body: " + string(readBody))
		log.WithFields(log.Fields{"UUID": conceptUUID}).Error(err)
		return err
	}

	return nil
}

/*
{
    "prefUUID": "11eecde3-c8ff-46a5-bd35-a5ef0598facd",
    "prefLabel": "Topic Name",
    "type": "Organisation",
    "sourceRepresentations": [
        {
            "uuid": "11eecde3-c8ff-46a5-bd35-a5ef0598facd",
            "prefLabel": "Nicky's Test Org 2",
            "authority": "TME",
            "authorityValue": "1234",
            "type": "Organisation"
        }
    ]
}
*/

type AggregatedConcept struct {
	PrefUUID              string    `json:"prefUUID,omitempty"`
	PrefLabel             string    `json:"prefLabel,omitempty"`
	Type                  string    `json:"type,omitempty"`
	SourceRepresentations []Concept `json:"sourceRepresentations,omitempty"`
}

// Concept - could be any concept genre, subject etc
type Concept struct {
	UUID           string `json:"uuid,omitempty"`
	PrefLabel      string `json:"prefLabel,omitempty"`
	Type           string `json:"type,omitempty"`
	Authority      string `json:"authority,omitempty"`
	AuthorityValue string `json:"authorityValue,omitempty"`
}

func getAggregatedConcept(uuid string, prefLabel string) AggregatedConcept {
	return AggregatedConcept{
		PrefLabel: prefLabel,
		PrefUUID:  uuid,
		Type:      "Topic",
		SourceRepresentations: []Concept{
			Concept{
				UUID:           uuid,
				PrefLabel:      prefLabel,
				Authority:      "Smartlogic",
				AuthorityValue: uuid,
				Type:           "Topic",
			},
		},
	}
}

func sendToConceptsRW(url string, conceptUUID string, prefLabel string, auth string) error {
	concept := getAggregatedConcept(conceptUUID, prefLabel)
	client := http.Client{}
	body, err := json.Marshal(concept)

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(concept)

	log.Infof("Sending Body: %v", b)

	if err != nil {
		return err
	}

	requestURL := url + "/" + conceptUUID

	request, _ := http.NewRequest("PUT", requestURL, strings.NewReader(string(body)))

	log.Infof("Sending Request: %v", request)

	request.Header.Set("authorization", auth)

	// -H 'content-type: application/ld+json' \
	request.Header.Set("content-type", "application/json")
	request.ContentLength = -1
	resp, reqErr := client.Do(request)

	defer resp.Body.Close()

	readBody, _ := ioutil.ReadAll(resp.Body)

	if reqErr != nil || resp.StatusCode/100 != 2 {
		err := errors.New("Request to " + requestURL + " returned status: " + strconv.Itoa(resp.StatusCode) + "; UUID: " + conceptUUID + " Response Body: " + string(readBody))
		log.WithFields(log.Fields{"UUID": conceptUUID}).Error(err)
		return err
	}

	return nil
}
