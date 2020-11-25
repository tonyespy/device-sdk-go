// -*- Mode: Go; indent-tabs-mode: t -*-
//
//
// SPDX-License-Identifier: Apache-2.0

PACKAGE AUTOEVENT

import (
	"context"
	"fmt"
	"sync"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/di"

	"github.com/edgexfoundry/device-sdk-go/internal/cache"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
)

type Manager interface {
	StartAutoEvents(dic *di.Container) bool
	StopAutoEvents()
	StopForDevice(deviceName string)
}

type manager struct {
	executorMap map[string][]Executor
	wg          *sync.WaitGroup
	mutex       sync.Mutex
}

var (
	createOnce sync.Onc
	m          *manager
)

// NewManager initiates the AutoEvent manager once
func NewManager(ctx context.Context, wg *sync.WaitGroup) {
	m = &manager{
		ctx:         ctx,
		wg:          wg,
		executorMap: make(map[string][]Executor)}
}

func (m *manager) StartAutoEvents(dic *di.Container) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	createOnce.Do(func() {
		for _, d := range cache.Devices().All() {
			if _, ok := m.executorMap[d.Name]; !ok {
				executors := m.triggerExecutors(d.Name, d.AutoEvents, dic)
				m.executorMap[d.Name] = executors
			}
		}
	})

	return true
}

// StopAutoEvents stops all the AutoEvents of the Device Service
func (m *manager) StopAutoEvents() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for deviceName, executors := range m.executorMap {
		for _, executor := range executors {
			executor.Stop()
		}
		delete(m.executorMap, deviceName)
	}
}

func (m *manager) triggerExecutors(deviceName string, autoEvents []contract.AutoEvent, dic *di.Container) []Executor {
	var executors []Executor
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	for _, autoEvent := range autoEvents {
		executor, err := NewExecutor(deviceName, autoEvent)
		if err != nil {
			lc.Error(fmt.Sprintf("AutoEvent for resource %s cannot be created, %v", autoEvent.Resource, err))
			// skip this AutoEvent if it causes error during creation
			continue
		}
		executors = append(executors, executor)
		go executor.Run(m.ctx, m.wg, dic)
	}
	return executors
}

// RestartForDevice restarts all the AutoEvents of the specific Device
func (m *manager) RestartForDevice(deviceName string, dic *di.Container) {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	m.StopForDevice(deviceName)
	d, ok := cache.Devices().ForName(deviceName)
	if !ok {
		lc.Error(fmt.Sprintf("there is no Device %s in cache to start AutoEvent", deviceName))
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	executors := m.triggerExecutors(deviceName, d.AutoEvents, dic)
	m.executorMap[deviceName] = executors
}

// StopForDevice stops all the AutoEvents of the specific Device
func (m *manager) StopForDevice(deviceName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	executors, ok := m.executorMap[deviceName]
	if ok {
		for _, executor := range executors {
			executor.Stop()
		}
		delete(m.executorMap, deviceName)
	}
}

// GetManager returns Manager instance
func GetManager() Manager {
	return m
}
