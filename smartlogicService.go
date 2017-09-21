package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bytes"
	"io/ioutil"

	uuidutils "github.com/Financial-Times/uuid-utils-go"
	log "github.com/sirupsen/logrus"
)

type SmartlogicService struct {
	smartlogicAPIKey  string
	smartlogicAddress string
	httpClient        http.Client
}

// Service defines the functions any read-write application needs to implement
type SmartlogicServicer interface {
	Write(prefLabel string) (conceptUUID string, err error)
}

//SmartlogicService instantiate driver
func NewSmartlogicService(smartlogicAPIKey string, smartlogicAddress string) SmartlogicService {
	return SmartlogicService{smartlogicAPIKey:smartlogicAPIKey, smartlogicAddress:smartlogicAddress}
}

/*
{
      "@type": [
		"skos:Concept",
		"http://www.ft.com/ontology/Topic"
      ],
      "skos:topConceptOf":{
    	"@id":"http://www.ft.com/thing/ConceptScheme/4fd43cbb-bd66-4825-86b8-f9a28d1bc366"
		},
      "skosxl:prefLabel": [
      {
         "@type":[
            "skosxl:Label"
         ],
         "skosxl:literalForm":[
            {
               "@language":"en",
               "@value":"New Topic Name2"
            }
         ]
    }
	]
}
*/

type Concept struct {
	Uri          string      `json:"@id"`
	Types        []string    `json:"@type,omitempty"`
	PrefLabels   []PrefLabel `json:"skosxl:prefLabel,omitempty"`
	TopConceptOf RefConcept  `json:"skos:topConceptOf,omitempty"`
	GUUID []SemGUUID `json:"sem:guid"`
}

type RefConcept struct {
	Uri   string     `json:"@id"`
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

func (s SmartlogicService) Write(prefLabel string) (string, error) {
	// TODO generate it randomly
	uuid := uuidutils.NewV3UUID(prefLabel).String()
	conceptToWrite := Concept{
		Uri: "http://www.ft.com/thing/" + uuid,
		Types: []string{
			"skos:Concept",
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

	log.WithFields(log.Fields{"UUID": uuid, "Concept": conceptToWrite}).Infof("Sending concept to smartlogic")
	sendToWriter(s.httpClient, s.smartlogicAddress, uuid, conceptToWrite, s.smartlogicAPIKey)
	return uuid, nil
}

func sendToWriter(client http.Client, url string, conceptUUID string, concept Concept, auth string) error {

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
