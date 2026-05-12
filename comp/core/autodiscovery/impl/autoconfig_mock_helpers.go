// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build test

package autodiscoveryimpl

import (
	autodiscoverydef "github.com/DataDog/datadog-agent/comp/core/autodiscovery/def"
	"github.com/DataDog/datadog-agent/comp/core/autodiscovery/scheduler"
	log "github.com/DataDog/datadog-agent/comp/core/log/def"
	secrets "github.com/DataDog/datadog-agent/comp/core/secrets/def"
	mockTagger "github.com/DataDog/datadog-agent/comp/core/tagger/mock"
	telemetry "github.com/DataDog/datadog-agent/comp/core/telemetry/def"
	workloadfilter "github.com/DataDog/datadog-agent/comp/core/workloadfilter/def"
	workloadmeta "github.com/DataDog/datadog-agent/comp/core/workloadmeta/def"
	healthplatformdef "github.com/DataDog/datadog-agent/comp/healthplatform/core/def"
	"github.com/DataDog/datadog-agent/pkg/util/option"
)

// NewAutoConfigForMock creates an AutoConfig instance for use in mocks.
func NewAutoConfigForMock(
	schedulerController *scheduler.Controller,
	secretResolver secrets.Component,
	wmeta option.Option[workloadmeta.Component],
	taggerComp mockTagger.Mock,
	logs log.Component,
	telemetryComp telemetry.Component,
	filterStore workloadfilter.Component,
	hp option.Option[healthplatformdef.Component],
) autodiscoverydef.Component {
	return createNewAutoConfig(schedulerController, secretResolver, wmeta, taggerComp, logs, telemetryComp, filterStore, hp)
}
