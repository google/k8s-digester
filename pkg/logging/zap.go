// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/google/k8s-digester/pkg/util"
)

// SyncLogger contains a logger and an associated sync function that should
// be called using `defer`.
type SyncLogger struct {
	Log  logr.Logger
	Sync func() error
}

// CreateZapLogger for structured logging
func CreateZapLogger(name string) (*SyncLogger, error) {
	zLog, err := createZapLogger()
	if err != nil {
		return nil, fmt.Errorf("could not create zap logger %w", err)
	}
	log := zapr.NewLogger(zLog).WithName(name)
	return &SyncLogger{
		Log:  log,
		Sync: zLog.Sync,
	}, nil
}

func createZapLogger() (*zap.Logger, error) {
	if util.IsDebug() {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
