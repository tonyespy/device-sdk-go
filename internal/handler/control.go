// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2017-2018 Canonical Ltd
// Copyright (C) 2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/edgexfoundry/device-sdk-go/internal/cache"
	"github.com/edgexfoundry/device-sdk-go/internal/common"
	"github.com/edgexfoundry/device-sdk-go/pkg/models"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/google/uuid"
)

var (
	id     string
	mtx    sync.Mutex
	busy   bool
)

func TransformHandler(requestMap map[string]string) (map[string]string, common.AppError) {
	common.LoggingClient.Info(fmt.Sprintf("service: transform request: transformData: %s", requestMap["transformData"]))
	return requestMap, nil
}

func DiscoveryHandler(w http.ResponseWriter) {
	// TODO: since var is set in two goroutines, it should be guarded
	// with the mutex as well...
	if id == "" {
		id = uuid.New().String()
	}

	if w != nil {
		msg := fmt.Sprintf("Discovery triggered or already running, id = %s", id)
		w.WriteHeader(http.StatusAccepted) //status=202
		_, _ = io.WriteString(w, msg)
	}

	defer mtx.Unlock()
	mtx.Lock()

	if busy {
		common.LoggingClient.Info(fmt.Sprintf("Discovery request returned. discovery process is running"))
		return
	}
	busy = true
	common.LoggingClient.Info(fmt.Sprintf("service %s discovery triggered", common.ServiceName))

	ctx := context.WithValue(context.Background(), common.CorrelationHeader, id)
	deviceCh := make(chan []models.DiscoveredDevice)
	go common.Discovery.Discover(deviceCh)
	go filterAndAddition(ctx, deviceCh)
}

func filterAndAddition(ctx context.Context, deviceCh <-chan []models.DiscoveredDevice) {
	pws := cache.ProvisionWatchers().All()
	devices := <-deviceCh

	mtx.Lock()
	busy = false
	mtx.Unlock()

	id = ""

	for _, d := range devices {
		for _, pw := range pws {
			if !whitelistPass(d, pw) {
				break
			}
			if !blacklistPass(d, pw) {
				break
			}

			if _, ok := cache.Devices().ForName(d.Name); ok {
				common.LoggingClient.Info(fmt.Sprintf("Candidate discovered device %s already existed", d.Name))
				break
			}

			common.LoggingClient.Info(fmt.Sprintf("Updating discovered device %s to Edgex", d.Name))
			millis := time.Now().UnixNano() / int64(time.Millisecond)
			device := &contract.Device{
				Name:           d.Name,
				Profile:        pw.Profile,
				Protocols:      d.Protocols,
				Labels:         d.Labels,
				Service:        pw.Service,
				AdminState:     pw.AdminState,
				OperatingState: contract.Enabled,
				AutoEvents:     nil,
			}
			device.Origin = millis
			device.Description = d.Description
			_, err := common.DeviceClient.Add(device, ctx)
			if err != nil {
				common.LoggingClient.Error(fmt.Sprintf("Created discovered device %s failed: %v", device.Name, err))
			}
		}
	}
	common.LoggingClient.Debug("Filtered device addition finished")
}

func whitelistPass(d models.DiscoveredDevice, pw contract.ProvisionWatcher) bool {
	// a candidate device should pass all identifiers
	for name, regex := range pw.Identifiers {
		// ignore the device protocol properties name
		for _, protocol := range d.Protocols {
			if value, ok := protocol[name]; ok {
				matched, err := regexp.MatchString(regex, value)
				if !matched || err != nil {
					common.LoggingClient.Debug(fmt.Sprintf("Device %s's %s value %s did not match PW identifier: %s", d.Name, name, value, regex))
					return false
				}
			} else {
				common.LoggingClient.Debug(fmt.Sprintf("Identifier field: %s, did not exist in discovered device %s", name, d.Name))
				return false
			}
		}
	}
	return true
}

func blacklistPass(d models.DiscoveredDevice, pw contract.ProvisionWatcher) bool {
	// a candidate should match none of the blocking identifiers
	for name, blacklist := range pw.BlockingIdentifiers {
		// ignore the device protocol properties name
		for _, protocol := range d.Protocols {
			if value, ok := protocol[name]; ok {
				for _, v := range blacklist {
					if value == v {
						common.LoggingClient.Debug(fmt.Sprintf("Discovered Device %s's %s should not be %s", d.Name, name, value))
						return false
					}
				}
			}
		}
	}
	return true
}
