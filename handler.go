package main

import (
	"fmt"
	"net/http"
	"net/url"
)

type RequestHandler struct {
	SmartlogicService SmartlogicServicer
	AnnotationsService AnnotationsServicer
}

func NewHandler(smartlogicService SmartlogicServicer, annotationsService AnnotationsServicer) RequestHandler {
	return RequestHandler{smartlogicService, annotationsService}
}


func (handler *RequestHandler) sendAnnotations(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	m, _ := url.ParseQuery(request.URL.RawQuery)

	_, exists := m["contentUUID"]

	if !exists {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(
			`{"message": "Missing or empty query parameter contentUUIDs."}`))
		return
	}
	contentUUIDs := m["contentUUID"]
	_, exists = m["conceptUUID"]
	if !exists {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(
			`{"message": "Missing or empty query parameter conceptUUID."}`))
		return
	}
	conceptUUID := m["conceptUUID"][0]

	err := handler.AnnotationsService.Write(conceptUUID, contentUUIDs)

	switch err {
	case nil:
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(
			fmt.Sprintf(`{"message":"Sent annotations to PAC for concept UUID: %v"}`, conceptUUID),
		))
	default:
		writer.WriteHeader(http.StatusInternalServerError)
	}

}

func (handler *RequestHandler) createTopic(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	m, _ := url.ParseQuery(request.URL.RawQuery)

	_, exists := m["prefLabel"]
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if !exists {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(
			`{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`))
		return
	}

	prefLabel := m["prefLabel"][0]

	uuid, err := handler.SmartlogicService.Write(prefLabel)

	switch err {
	case nil:
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(
			fmt.Sprintf(`{"conceptUUID":"%v"}`, uuid),
		))
	default:
		writer.WriteHeader(http.StatusInternalServerError)
	}

}
