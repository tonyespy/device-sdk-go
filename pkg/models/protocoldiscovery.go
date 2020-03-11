// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package models

import contract "github.com/edgexfoundry/go-mod-core-contracts/models"

// ProtocolDiscovery is a low-level device-specific interface implemented
// by device services that support dynamic device discovery.
type ProtocolDiscovery interface {
	// Discover triggers protocol specific device discovery, which is
	// an asynchronous operation, returns a list of new devices
	// that may be added to the device service based on a set of acceptance
	// criteria (i.e. Provision Watchers).
	Discover(devices chan<- []DiscoveredDevice)
}

// DiscoveredDevice defines the required information for a found device.
type DiscoveredDevice struct {
	Name        string
	Protocols   map[string]contract.ProtocolProperties
	Description string
	Labels      []string
}
