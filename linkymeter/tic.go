// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package linkymeter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

var ErrTIC2WebsocketNotConnected = errors.New("tic2websocket not connected")
var ErrTIC2WebsocketAlreadyConnected = errors.New("tic2websocket already connected")

type OnTICData func(ticData TICData)
type OnTICError func(ticError TICError)
type OnTICAbnormalClosure func()

type TICEventSubscriber struct {
	onData            OnTICData
	onError           OnTICError
	onAbnormalClosure OnTICAbnormalClosure
}

type TICSubsciberMapEntry struct {
	ticIdentifier   TICIdentifier
	eventSubscriber TICEventSubscriber
}

type TIC2WebsocketClient struct {
	url                     url.URL
	connection              *websocket.Conn
	connected               atomic.Bool
	subscriberLock          sync.Mutex
	subscriberMap           map[string]TICSubsciberMapEntry
	readMessagesRunning     atomic.Bool
	stopReadMessagesChannel chan bool
	responseChannel         chan TIC2WebsocketResponse
}

type TICIdentifier struct {
	SerialNumber string `json:"serialNumber,omitempty"`
	PortName     string `json:"portName,omitempty"`
	PortId       string `json:"portId,omitempty"`
}

type ModemInfo struct {
	PortName         string `json:"portName"`
	ModemType        string `json:"modemType,omitempty"`
	ProductId        int    `json:"productId,omitempty"`
	VendorId         int    `json:"vendorId,omitempty"`
	ProductName      string `json:"productName,omitempty"`
	ManufacturerName string `json:"manufacturerName,omitempty"`
	SerialNumber     string `json:"serialNumber,omitempty"`
	PortId           string `json:"portId,omitempty"`
}

type TICData struct {
	Mode            string            `json:"mode"`
	CaptureDateTime string            `json:"captureDateTime"`
	Identifier      TICIdentifier     `json:"identifier"`
	Content         map[string]string `json:"content"`
}

type TICError struct {
	ErrorCode    int           `json:"errorCode"`
	ErrorMessage string        `json:"errorMessage"`
	Identifier   TICIdentifier `json:"identifier"`
}

type TIC2WebsocketRequest struct {
	Name string      `json:"name"`
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type TIC2WebsocketResponse struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	DateTime     string      `json:"dateTime"`
	ErrorCode    int         `json:"errorCode"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	Data         interface{} `json:"data,omitempty"`
}

type TIC2WebsocketEvent struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	DateTime string                 `json:"dateTime"`
	Data     map[string]interface{} `json:"data"`
}

func IsEmptyIdentifier(identifier TICIdentifier) bool {
	return len(identifier.PortId) == 0 && len(identifier.PortName) == 0 && len(identifier.SerialNumber) == 0
}

func NewTIC2WebsocketClient() *TIC2WebsocketClient {
	client := &TIC2WebsocketClient{}

	client.subscriberMap = make(map[string]TICSubsciberMapEntry)
	client.stopReadMessagesChannel = make(chan bool)
	client.responseChannel = make(chan TIC2WebsocketResponse)

	return client
}

func (t *TIC2WebsocketClient) Connect(host string) error {
	if t.connected.Load() {
		return ErrTIC2WebsocketAlreadyConnected
	}
	url := url.URL{Scheme: "ws", Host: host, Path: "/"}
	connection, response, error := websocket.DefaultDialer.Dial(url.String(), nil)
	if error != nil {
		if response == nil {
			return fmt.Errorf("connexion to %s failed : %s", url.String(), error.Error())
		} else {
			return fmt.Errorf("connexion to %s failed with status code %d : %s", url.String(), response.StatusCode, error.Error())
		}
	}
	t.url = url
	t.connection = connection
	t.connected.Store(true)

	go t.readMessages()

	return nil
}

func (t *TIC2WebsocketClient) IsConnected() bool {
	return t.connected.Load()
}

func (t *TIC2WebsocketClient) GetAvailableTICs() ([]TICIdentifier, error) {
	var availableTICs []TICIdentifier

	response, error := t.executeTransaction(TIC2WebsocketRequest{Type: "REQUEST", Name: "GetAvailableTICs"})
	if error != nil {
		return []TICIdentifier{}, error
	}
	error = mapstructure.Decode(response.Data, &availableTICs)

	return availableTICs, error
}

func (t *TIC2WebsocketClient) GetModemsInfo() ([]ModemInfo, error) {
	var modemsInfo []ModemInfo

	response, error := t.executeTransaction(TIC2WebsocketRequest{Type: "REQUEST", Name: "GetModemsInfo"})
	if error != nil {
		return []ModemInfo{}, error
	}
	error = mapstructure.Decode(response.Data, &modemsInfo)

	return modemsInfo, error
}

func (t *TIC2WebsocketClient) ReadTIC(identifier TICIdentifier) (TICData, error) {
	var ticData TICData

	response, error := t.executeTransaction(TIC2WebsocketRequest{Type: "REQUEST", Name: "ReadTIC", Data: identifier})
	if error != nil {
		return TICData{}, error
	}
	error = mapstructure.Decode(response.Data, &ticData)

	return ticData, error
}

func (t *TIC2WebsocketClient) SubscribeTIC(onData OnTICData, onError OnTICError, onAbnormalClosure OnTICAbnormalClosure, ticIdentifier TICIdentifier) (string, error) {
	if IsEmptyIdentifier(ticIdentifier) {
		return "", fmt.Errorf("cannot subscribe with empty identifier")
	}
	request := TIC2WebsocketRequest{Type: "REQUEST", Name: "SubscribeTIC", Data: ticIdentifier}

	_, error := t.executeTransaction(request)
	if error != nil {
		return "", error
	}
	subscriptionId := uuid.New().String()
	subscriber := TICEventSubscriber{onData: onData, onError: onError, onAbnormalClosure: onAbnormalClosure}
	t.subscriberLock.Lock()
	entry := TICSubsciberMapEntry{ticIdentifier: ticIdentifier, eventSubscriber: subscriber}
	t.subscriberMap[subscriptionId] = entry
	t.subscriberLock.Unlock()

	return subscriptionId, nil
}

func (t *TIC2WebsocketClient) UnsubscribeTIC(subscriptionId string) error {
	_, ok := t.subscriberMap[subscriptionId]
	if !ok {
		return fmt.Errorf("subscriber id '%s' not found", subscriptionId)
	}
	_, error := t.executeTransaction(TIC2WebsocketRequest{Type: "REQUEST", Name: "UnsubscribeTIC"})
	if error != nil {
		return error
	}
	t.subscriberLock.Lock()
	delete(t.subscriberMap, subscriptionId)
	t.subscriberLock.Unlock()

	return nil
}

func (t *TIC2WebsocketClient) GetSubscribers(ticIdentifier TICIdentifier) map[string]TICIdentifier {
	subscriberIdMap := make(map[string]TICIdentifier)

	t.subscriberLock.Lock()
	for id, entry := range t.subscriberMap {
		if cmp.Equal(entry.ticIdentifier, ticIdentifier) {
			subscriberIdMap[id] = entry.ticIdentifier
		}
	}
	t.subscriberLock.Unlock()

	return subscriberIdMap
}

func (t *TIC2WebsocketClient) CheckSubscriber(subscriptionId string, ticIdentifier TICIdentifier) bool {
	subscriberIdMap := t.GetSubscribers(ticIdentifier)
	if len(subscriberIdMap) > 0 {
		for id := range subscriberIdMap {
			if id == subscriptionId {
				return true
			}
		}
	}

	return false
}

func (t *TIC2WebsocketClient) readMessages() {
	log.Debug("Start readMessages")
	t.readMessagesRunning.Store(true)
	for {
		select {
		case <-t.stopReadMessagesChannel:
			log.Debug("Stop readMessages")
			t.readMessagesRunning.Store(false)
			return
		default:
			var message map[string]interface{}
			_, messageBytes, error := t.connection.ReadMessage()
			if error != nil {
				log.Errorf("cannot read message : %s", error.Error())
				if websocket.IsCloseError(error, websocket.CloseAbnormalClosure) {
					t.readMessagesRunning.Store(false)
					t.onAbnormalClosure()
					return
				}
				continue
			}
			log.Tracef("Message  : %s", string(messageBytes))
			error = json.Unmarshal(messageBytes, &message)
			if error != nil {
				log.Errorf("cannot decode message : %s", error.Error())
				continue
			}
			if message["type"] == "RESPONSE" {
				var response TIC2WebsocketResponse
				error = json.Unmarshal(messageBytes, &response)
				if error != nil {
					log.Errorf("cannot decode response from message %+v : %s", message, error.Error())
					continue
				}
				log.Tracef("Response : %+v\n", response)
				t.responseChannel <- response
			} else if message["type"] == "EVENT" {
				var event TIC2WebsocketEvent
				error = json.Unmarshal(messageBytes, &event)
				if error != nil {
					log.Errorf("cannot decode event from message %+v : %s", message, error.Error())
					continue
				}
				log.Tracef("Event    : %+v\n", event)
				var dataBytes []byte
				dataBytes, error = json.Marshal(event.Data)
				if error != nil {
					log.Errorf("cannot decode event data : %s", error.Error())
					continue
				}
				if event.Name == "OnTICData" {
					var ticData TICData
					error = json.Unmarshal(dataBytes, &ticData)
					if error != nil {
						log.Errorf("cannot decode ticData from event %+v : %s", event, error.Error())
						continue
					}
					t.subscriberLock.Lock()
					for _, entry := range t.subscriberMap {
						go entry.eventSubscriber.onData(ticData)
					}
					t.subscriberLock.Unlock()
				} else if event.Name == "OnError" {
					var ticError TICError
					error = json.Unmarshal(dataBytes, &ticError)
					if error != nil {
						log.Errorf("cannot decode ticError from event %+v : %s", event, error.Error())
						continue
					}
					t.subscriberLock.Lock()
					for _, entry := range t.subscriberMap {
						go entry.eventSubscriber.onError(ticError)
					}
					t.subscriberLock.Unlock()
				}
			}
		}
	}
}

func (t *TIC2WebsocketClient) Close() error {
	if !t.connected.Load() {
		return ErrTIC2WebsocketNotConnected
	}
	deadline := time.Now().Add(1 * time.Second)
	error := t.connection.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		deadline,
	)
	if error != nil {
		return fmt.Errorf("sending close message to server failed : %s", error.Error())
	}
	error = t.connection.SetReadDeadline(deadline)
	if error != nil {
		return fmt.Errorf("set read dead line failed : %s", error.Error())
	}
	if t.readMessagesRunning.Load() {
		log.Debug("stopping readMessages")
		t.stopReadMessagesChannel <- true
	}
	error = t.connection.Close()
	if error != nil {
		return fmt.Errorf("close connection failed : %s", error.Error())
	}
	t.connected.Store(false)

	return error
}

func (t *TIC2WebsocketClient) onAbnormalClosure() {
	log.Error("abnormal closure detected")
	t.subscriberLock.Lock()
	for _, entry := range t.subscriberMap {
		go entry.eventSubscriber.onAbnormalClosure()
	}
	t.subscriberLock.Unlock()
	err := t.connection.Close()
	if err != nil {
		log.Errorf("close connection failed : %s", err.Error())
	}
	t.connected.Store(false)
}

func (t *TIC2WebsocketClient) executeTransaction(request TIC2WebsocketRequest) (TIC2WebsocketResponse, error) {
	if !t.connected.Load() {
		return TIC2WebsocketResponse{}, ErrTIC2WebsocketNotConnected
	}
	requestBytes, error := json.Marshal(request)
	if error != nil {
		return TIC2WebsocketResponse{}, fmt.Errorf("cannot encode %s request : %s", request.Name, error.Error())
	}
	log.Tracef("Request  : %s\n", string(requestBytes))
	error = t.connection.WriteMessage(websocket.TextMessage, requestBytes)
	if error != nil {
		return TIC2WebsocketResponse{}, fmt.Errorf("cannot send %s request : %s", request.Name, error.Error())
	}
	processTimeout := time.After(10 * time.Second)
	select {
	case <-processTimeout:
		return TIC2WebsocketResponse{}, fmt.Errorf("%s timeout", request.Name)
	case response := <-t.responseChannel:
		if response.ErrorCode != 0 {
			return response, fmt.Errorf("%s failed : %s", request.Name, response.ErrorMessage)
		}
		return response, nil
	}
}
