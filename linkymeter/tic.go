// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

/*
Package linkymeter implements utility routines for receiving and computing Linky meter data needed by the energy management system.

The reception of Linky meter data is done through a TIC2WebSocket client.
This client receives data periodically (from 1 to 3 seconds) and a routine used by the energy management system computes data to be stored in the data model.
Those data are then used by the energy management system to decide the electrical vehicle charge power to be sent to the wallbox.

See [MeterData] for the data provided by the Linky meter.
*/
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

var errTic2WebsocketNotConnected = errors.New("tic2websocket not connected")
var errTic2WebsocketAlreadyConnected = errors.New("tic2websocket already connected")

// Callback function used to notify new data received from the TIC2WebSocket client
type OnTicData func(ticData TicData)

// Callback function used to notify an error received from the TIC2WebSocket client
type OnTicError func(ticError TicError)

// Callback function used to notify an abnormal closure of the TIC2WebSocket client
type OnTicAbnormalClosure func()

type ticEventSubscriber struct {
	onData            OnTicData
	onError           OnTicError
	onAbnormalClosure OnTicAbnormalClosure
}

type ticSubsciberMapEntry struct {
	ticIdentifier   TicIdentifier
	eventSubscriber ticEventSubscriber
}

// Tic2WebsocketClient handle the connection to the TIC2WebSocket server
type Tic2WebsocketClient struct {
	url                     url.URL
	connection              *websocket.Conn
	connected               atomic.Bool
	subscriberLock          sync.Mutex
	subscriberMap           map[string]ticSubsciberMapEntry
	readMessagesRunning     atomic.Bool
	stopReadMessagesChannel chan bool
	responseChannel         chan tic2WebsocketResponse
}

// TicIdentifier identifies a TIC to be read or subscribed
type TicIdentifier struct {
	// TIC serial number (if empty, the first available TIC is used)
	SerialNumber string `json:"serialNumber,omitempty"`
	// TIC port name (if empty, the first available TIC is used)
	PortName string `json:"portName,omitempty"`
	// TIC port ID (if empty, the first available TIC is used)
	PortId string `json:"portId,omitempty"`
}

// ModemInfo represents the TIC modem information
type ModemInfo struct {
	// TIC serial port name
	PortName string `json:"portName"`
	// TIC modem type (e.g. "MICHAUD", "TELEINFO" for USB modems)
	ModemType string `json:"modemType,omitempty"`
	// TIC modem USB product ID
	ProductId int `json:"productId,omitempty"`
	// TIC modem USB vendor ID
	VendorId int `json:"vendorId,omitempty"`
	// TIC modem USB product name
	ProductName string `json:"productName,omitempty"`
	// TIC modem USB manufacturer name
	ManufacturerName string `json:"manufacturerName,omitempty"`
	// TIC modem USB serial number
	SerialNumber string `json:"serialNumber,omitempty"`
	// TIC modem USB port ID
	PortId string `json:"portId,omitempty"`
}

// TicData represents the TIC data read or received from the TIC2WebSocket server
type TicData struct {
	// TIC format (e.g. "STANDARD", "HISTORIC")
	Mode string `json:"mode"`
	// TIC capture date time (format "2006-01-02T15:04:05.000Z07:00")
	CaptureDateTime string `json:"captureDateTime"`
	// TIC identifier associated to the data
	Identifier TicIdentifier `json:"identifier"`
	// TIC raw content (map of TIC data name and value)
	Content map[string]string `json:"content"`
}

// TicError represents an error received from the TIC2WebSocket server
type TicError struct {
	// Error code (0 if no error)
	ErrorCode int `json:"errorCode"`
	// Error message (empty if no error)
	ErrorMessage string `json:"errorMessage"`
	// TIC identifier associated to the error
	Identifier TicIdentifier `json:"identifier"`
}

type tic2WebsocketRequest struct {
	Name string      `json:"name"`
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type tic2WebsocketResponse struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	DateTime     string      `json:"dateTime"`
	ErrorCode    int         `json:"errorCode"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	Data         interface{} `json:"data,omitempty"`
}

type tic2WebsocketEvent struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	DateTime string                 `json:"dateTime"`
	Data     map[string]interface{} `json:"data"`
}

// IsEmptyIdentifier returns true if the TIC identifier is empty, false otherwise
func IsEmptyIdentifier(identifier TicIdentifier) bool {
	return len(identifier.PortId) == 0 && len(identifier.PortName) == 0 && len(identifier.SerialNumber) == 0
}

// NewTic2WebsocketClient creates a new TIC2WebSocket client
func NewTic2WebsocketClient() *Tic2WebsocketClient {
	client := &Tic2WebsocketClient{}

	client.subscriberMap = make(map[string]ticSubsciberMapEntry)
	client.stopReadMessagesChannel = make(chan bool)
	client.responseChannel = make(chan tic2WebsocketResponse)

	return client
}

// Connect connects to the TIC2WebSocket server at the given host
func (t *Tic2WebsocketClient) Connect(host string) error {
	if t.connected.Load() {
		return errTic2WebsocketAlreadyConnected
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

// IsConnected returns true if the connection with the TIC2WebSocket server is established, false otherwise
func (t *Tic2WebsocketClient) IsConnected() bool {
	return t.connected.Load()
}

// GetAvailableTics returns the list of available TICs stream from the TIC2WebSocket server
func (t *Tic2WebsocketClient) GetAvailableTics() ([]TicIdentifier, error) {
	var availableTICs []TicIdentifier

	response, error := t.executeTransaction(tic2WebsocketRequest{Type: "REQUEST", Name: "GetAvailableTICs"})
	if error != nil {
		return []TicIdentifier{}, error
	}
	error = mapstructure.Decode(response.Data, &availableTICs)

	return availableTICs, error
}

// GetModemsInfo returns the list of TIC modems information from the TIC2WebSocket server
func (t *Tic2WebsocketClient) GetModemsInfo() ([]ModemInfo, error) {
	var modemsInfo []ModemInfo

	response, error := t.executeTransaction(tic2WebsocketRequest{Type: "REQUEST", Name: "GetModemsInfo"})
	if error != nil {
		return []ModemInfo{}, error
	}
	error = mapstructure.Decode(response.Data, &modemsInfo)

	return modemsInfo, error
}

// ReadTic reads the TIC data identified by the given TIC identifier from the TIC2WebSocket server
func (t *Tic2WebsocketClient) ReadTic(identifier TicIdentifier) (TicData, error) {
	var ticData TicData

	response, error := t.executeTransaction(tic2WebsocketRequest{Type: "REQUEST", Name: "ReadTIC", Data: identifier})
	if error != nil {
		return TicData{}, error
	}
	error = mapstructure.Decode(response.Data, &ticData)

	return ticData, error
}

// SubscribeTic subscribes to the TIC data identified by the given TIC identifier from the TIC2WebSocket server and returns a subscription ID
func (t *Tic2WebsocketClient) SubscribeTic(onData OnTicData, onError OnTicError, onAbnormalClosure OnTicAbnormalClosure, ticIdentifier TicIdentifier) (string, error) {
	if IsEmptyIdentifier(ticIdentifier) {
		return "", fmt.Errorf("cannot subscribe with empty identifier")
	}
	request := tic2WebsocketRequest{Type: "REQUEST", Name: "SubscribeTIC", Data: ticIdentifier}

	_, error := t.executeTransaction(request)
	if error != nil {
		return "", error
	}
	subscriptionId := uuid.New().String()
	subscriber := ticEventSubscriber{onData: onData, onError: onError, onAbnormalClosure: onAbnormalClosure}
	t.subscriberLock.Lock()
	entry := ticSubsciberMapEntry{ticIdentifier: ticIdentifier, eventSubscriber: subscriber}
	t.subscriberMap[subscriptionId] = entry
	t.subscriberLock.Unlock()

	return subscriptionId, nil
}

// UnsubscribeTic unsubscribes from the TIC data updates using the subscription ID
func (t *Tic2WebsocketClient) UnsubscribeTic(subscriptionId string) error {
	_, ok := t.subscriberMap[subscriptionId]
	if !ok {
		return fmt.Errorf("subscriber id '%s' not found", subscriptionId)
	}
	_, error := t.executeTransaction(tic2WebsocketRequest{Type: "REQUEST", Name: "UnsubscribeTIC"})
	if error != nil {
		return error
	}
	t.subscriberLock.Lock()
	delete(t.subscriberMap, subscriptionId)
	t.subscriberLock.Unlock()

	return nil
}

// GetSubscribers returns the map of subscription IDs and their TIC identifiers for the given TIC identifier
func (t *Tic2WebsocketClient) GetSubscribers(ticIdentifier TicIdentifier) map[string]TicIdentifier {
	subscriberIdMap := make(map[string]TicIdentifier)

	t.subscriberLock.Lock()
	for id, entry := range t.subscriberMap {
		if cmp.Equal(entry.ticIdentifier, ticIdentifier) {
			subscriberIdMap[id] = entry.ticIdentifier
		}
	}
	t.subscriberLock.Unlock()

	return subscriberIdMap
}

// CheckSubscriber checks if the given subscription ID is subscribed to the given TIC identifier
func (t *Tic2WebsocketClient) CheckSubscriber(subscriptionId string, ticIdentifier TicIdentifier) bool {
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

func (t *Tic2WebsocketClient) readMessages() {
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
				var response tic2WebsocketResponse
				error = json.Unmarshal(messageBytes, &response)
				if error != nil {
					log.Errorf("cannot decode response from message %+v : %s", message, error.Error())
					continue
				}
				log.Tracef("Response : %+v\n", response)
				t.responseChannel <- response
			} else if message["type"] == "EVENT" {
				var event tic2WebsocketEvent
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
					var ticData TicData
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
					var ticError TicError
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

// Close closes the connection to the TIC2WebSocket server
func (t *Tic2WebsocketClient) Close() error {
	if !t.connected.Load() {
		return errTic2WebsocketNotConnected
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

func (t *Tic2WebsocketClient) onAbnormalClosure() {
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

func (t *Tic2WebsocketClient) executeTransaction(request tic2WebsocketRequest) (tic2WebsocketResponse, error) {
	if !t.connected.Load() {
		return tic2WebsocketResponse{}, errTic2WebsocketNotConnected
	}
	requestBytes, error := json.Marshal(request)
	if error != nil {
		return tic2WebsocketResponse{}, fmt.Errorf("cannot encode %s request : %s", request.Name, error.Error())
	}
	log.Tracef("Request  : %s\n", string(requestBytes))
	error = t.connection.WriteMessage(websocket.TextMessage, requestBytes)
	if error != nil {
		return tic2WebsocketResponse{}, fmt.Errorf("cannot send %s request : %s", request.Name, error.Error())
	}
	processTimeout := time.After(10 * time.Second)
	select {
	case <-processTimeout:
		return tic2WebsocketResponse{}, fmt.Errorf("%s timeout", request.Name)
	case response := <-t.responseChannel:
		if response.ErrorCode != 0 {
			return response, fmt.Errorf("%s failed : %s", request.Name, response.ErrorMessage)
		}
		return response, nil
	}
}
