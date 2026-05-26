// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows

// Package installinfo offers helpers to interact with the 'install_info' file.
//
// The install_info files is present next to the agent configuration and contains information about how the agent was//
// installed and its version history.  The file is automatically updated by installation tools (MSI installer, Chef,
// Ansible, DPKG, ...).
package installinfo

import (
	"path/filepath"
)

var (
	configDir       = "/etc/datadog-agent"
	installInfoFile = filepath.Join(configDir, "install_info")
	installSigFile  = filepath.Join(configDir, "install.json")
)

// WriteInstallInfo is a no-op in this build.
//
// The agent no longer writes to /etc/datadog-agent at runtime. install_info
// and install.json must be provisioned by the packaging layer (deb/rpm/msi
// postinst, container image, or configuration management).
func WriteInstallInfo(_, _, _ string) error {
	return nil
}

// RmInstallInfo is a no-op in this build. See WriteInstallInfo.
func RmInstallInfo() {}
