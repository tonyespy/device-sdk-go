// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package autodiscovery

import (
	"time"

	"github.com/edgexfoundry/device-sdk-go/internal/common"
	"github.com/edgexfoundry/device-sdk-go/internal/handler"
)

func Run() {
	duration := common.CurrentConfig.Device.Discovery.Interval
	enabled := common.CurrentConfig.Device.Discovery.Enabled

	for {
		if duration <= 0 || !enabled {
			break
		}
		time.Sleep(time.Second * time.Duration(duration))

		common.LoggingClient.Debug("Auto-discovery triggered")
		handler.DiscoveryHandler(nil)
	}
}
