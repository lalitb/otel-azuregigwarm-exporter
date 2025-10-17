// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo
/* +build cgo */

#include "geneva_ffi.h"

/* Helpers to set union fields from Go (cgo cannot assign union fields directly) */

void geneva_set_cert(GenevaConfig* cfg, const char* path, const char* password) {
    if (!cfg) return;
    cfg->auth.cert.cert_path = path;
    cfg->auth.cert.cert_password = password;
}

void geneva_set_workload_identity(GenevaConfig* cfg, const char* resource) {
    if (!cfg) return;
    cfg->auth.workload_identity.resource = resource;
}