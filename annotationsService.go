package main

import (
	"net/http"
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"errors"
	"strconv"
	"fmt"
	"bytes"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
)

type Annotations struct {
	UUID string `json:"uuid"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type AnnotationsService struct {
	apiKey string
	httpClient        http.Client
}

// Service defines the functions any read-write application needs to implement
type AnnotationsServicer interface {
	Write(conceptUUID string, contentUUIDs []string) error
}

//AnnotationsService instantiate driver
func NewAnnotationsService(apiKey string) AnnotationsService {
	return AnnotationsService{apiKey:apiKey}
}

type Annotation struct {
	Predicate string
	UUID string `json:"id,omitempty"`
}

func (annotationService AnnotationsService) Write(conceptId string, contentUUIDs []string) error {
	for _, contentUUID := range contentUUIDs {
		annotations := []Annotation {
			Annotation{
				Predicate: "http://www.ft.com/ontology/annotation/about",
				UUID: mapper.IDURL(conceptId),
			},
		}
		existingAnnotations, err := getExistingAnnotations(annotationService.httpClient, contentUUID, annotationService.apiKey)

		if err != nil {
			return err
		}

		for _, annotation := range existingAnnotations {
			annotations = append(annotations, annotation)
		}

		log.Infof("Sending to PAC annotations: %v \n ContentUUID: %v", annotations, contentUUID)

		allAnnotations := Annotations{contentUUID, annotations}

		err = sendToPAC(annotationService.httpClient, contentUUID, allAnnotations, annotationService.apiKey)
		if err != nil {
			return err
		}
	}

	return nil

}

func getExistingAnnotations(client http.Client, contentUUID string, apiKey string) ([]Annotation, error){
	reqURL := fmt.Sprintf("http://test.api.ft.com/content/%v/annotations", contentUUID)
	request, _ := http.NewRequest("GET", reqURL, nil)
	log.Infof("Sending Request: %v", request)

	request.Header.Set("X-Api-Key", apiKey)
	request.Header.Set( "X-Request-Id", "hackothon1111")
	resp, reqErr := client.Do(request)

	log.Infof("Status Code: %v", reqErr)

	if reqErr != nil || resp.StatusCode > 202 {
		readBody, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		err := errors.New("Request to " + reqURL + " returned status: " + strconv.Itoa(resp.StatusCode) + "; Content UUID: "+ contentUUID + " Response Body: " + string(readBody))
		log.WithFields(log.Fields{"UUID":  contentUUID}).Error(err)
		return nil, err
	}

	var s = []Annotation{}
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &s)

	if err != nil {
		return nil, err
	}

	return s, nil
}

func sendToPAC(client http.Client, contentUUID string, annotations Annotations, apiKey string) error {

	body, err := json.Marshal(annotations)

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(annotations)

	log.Infof("Sending Annotations: %v", b)


	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("http://test.api.ft.com/drafts/content/%v/annotations/publish", contentUUID)
	request, _ := http.NewRequest("POST", reqURL, strings.NewReader(string(body)))

	log.Infof("Sending Request: %v", request)
	request.Header.Set("content-type", "application/json")
	request.Header.Set( "X-Request-Id", "hackothon1234")

	request.Header.Set("X-Api-Key", apiKey)
	request.ContentLength = -1
	resp, reqErr := client.Do(request)

	log.Infof("Status Code: %v", resp.StatusCode)

	if reqErr != nil || resp.StatusCode > 202 {
		readBody, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		err := errors.New("Request to " + reqURL + " returned status: " + strconv.Itoa(resp.StatusCode) + "; Content UUID: "+ contentUUID + " Response Body: " + string(readBody))
		log.WithFields(log.Fields{"UUID":  contentUUID}).Error(err)
		return err
	}

	return nil
}


