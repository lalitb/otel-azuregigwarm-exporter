// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package cgo

/*
#cgo CFLAGS: -I./headers
#cgo LDFLAGS: -L../../geneva_ffi_bridge/target/release -lgeneva_ffi_bridge
#include "headers/geneva_errors.h"
#include "headers/geneva_ffi.h"
#include <stdint.h>
#include <stdlib.h>

// Helpers to set union fields from Go (implemented in c_helpers.c)
void geneva_set_cert(GenevaConfig* cfg, const char* path, const char* password);
void geneva_set_workload_identity(GenevaConfig* cfg, const char* resource);
*/
import "C"
import (
	"errors"
	"fmt"
        "log"
	"runtime"
	"unsafe"
)

// GenevaClient wraps the Rust Geneva client handle
type GenevaClient struct {
	handle *C.GenevaClientHandle
}

// GenevaConfig represents the Geneva client configuration
type GenevaConfig struct {
	Endpoint           string
	Environment        string
	Account            string
	Namespace          string
	Region             string
	ConfigMajorVersion uint32
	AuthMethod         int32 // 0 = MSI, 1 = Certificate, 2 = WorkloadIdentity
	Tenant             string
	RoleName           string
	RoleInstance       string
	CertPath           string // Only used when AuthMethod == 1
	CertPassword       string // Only used when AuthMethod == 1
	WorkloadIdentityResource string // Only used when AuthMethod == 2
}

// GenevaError represents the error codes from the Rust FFI
type GenevaError C.GenevaError

const (
	GenevaSuccess              = GenevaError(C.GENEVA_SUCCESS)
	GenevaInvalidConfig        = GenevaError(C.GENEVA_INVALID_CONFIG)
	GenevaInitializationFailed = GenevaError(C.GENEVA_INITIALIZATION_FAILED)
	GenevaUploadFailed         = GenevaError(C.GENEVA_UPLOAD_FAILED)
	GenevaInvalidData          = GenevaError(C.GENEVA_INVALID_DATA)
	GenevaInternalError        = GenevaError(C.GENEVA_INTERNAL_ERROR)
)

// Add Go-typed constants for granular C error codes to avoid brittle numeric literals.
const (
	genevaErrNullPointer         C.GenevaError = C.GenevaError(100)
	genevaErrEmptyInput          C.GenevaError = C.GenevaError(101)
	genevaErrDecodeFailed        C.GenevaError = C.GenevaError(102)
	genevaErrIndexOutOfRange     C.GenevaError = C.GenevaError(103)
	genevaErrInvalidAuthMethod   C.GenevaError = C.GenevaError(110)
	genevaErrInvalidCertConfig   C.GenevaError = C.GenevaError(111)
	genevaErrMissingEndpoint     C.GenevaError = C.GenevaError(130)
	genevaErrMissingEnvironment  C.GenevaError = C.GenevaError(131)
	genevaErrMissingAccount      C.GenevaError = C.GenevaError(132)
	genevaErrMissingNamespace    C.GenevaError = C.GenevaError(133)
	genevaErrMissingRegion       C.GenevaError = C.GenevaError(134)
	genevaErrMissingTenant       C.GenevaError = C.GenevaError(135)
	genevaErrMissingRoleName     C.GenevaError = C.GenevaError(136)
	genevaErrMissingRoleInstance C.GenevaError = C.GenevaError(137)
)

// Error returns the string representation of the Geneva error
func (e GenevaError) Error() string {
	switch e {
	case GenevaSuccess:
		return "success"
	case GenevaInvalidConfig:
		return "invalid configuration"
	case GenevaInitializationFailed:
		return "initialization failed"
	case GenevaUploadFailed:
		return "upload failed"
	case GenevaInvalidData:
		return "invalid data"
	case GenevaInternalError:
		return "internal error"
	default:
		return fmt.Sprintf("unknown error: %d", int(e))
	}
}

// NewGenevaClient creates a new Geneva client using the Rust FFI
func NewGenevaClient(config GenevaConfig) (*GenevaClient, error) {
	// Convert Go strings to C strings
	cEndpoint := C.CString(config.Endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))

	cEnvironment := C.CString(config.Environment)
	defer C.free(unsafe.Pointer(cEnvironment))

	cAccount := C.CString(config.Account)
	defer C.free(unsafe.Pointer(cAccount))

	cNamespace := C.CString(config.Namespace)
	defer C.free(unsafe.Pointer(cNamespace))

	cRegion := C.CString(config.Region)
	defer C.free(unsafe.Pointer(cRegion))

	cTenant := C.CString(config.Tenant)
	defer C.free(unsafe.Pointer(cTenant))

	cRoleName := C.CString(config.RoleName)
	defer C.free(unsafe.Pointer(cRoleName))

	cRoleInstance := C.CString(config.RoleInstance)
	defer C.free(unsafe.Pointer(cRoleInstance))

	var cCertPath *C.char
	var cCertPassword *C.char
    var cWorkloadIdentityResource *C.char

	if config.AuthMethod == 1 { // Certificate auth
		cCertPath = C.CString(config.CertPath)
		defer C.free(unsafe.Pointer(cCertPath))

		cCertPassword = C.CString(config.CertPassword)
		defer C.free(unsafe.Pointer(cCertPassword))
	} else if config.AuthMethod == 2 { // Workload Identity auth
		cWorkloadIdentityResource = C.CString(config.WorkloadIdentityResource)
		defer C.free(unsafe.Pointer(cWorkloadIdentityResource))
	}

	// Create C config struct
	cConfig := C.GenevaConfig{
		endpoint:             cEndpoint,
		environment:          cEnvironment,
		account:              cAccount,
		namespace_name:       cNamespace,
		region:               cRegion,
		config_major_version: C.uint32_t(config.ConfigMajorVersion),
		auth_method:          C.uint32_t(config.AuthMethod),
		tenant:               cTenant,
		role_name:            cRoleName,
		role_instance:        cRoleInstance,
		msi_resource:         nil, // Optional MSI resource, not currently used
	}

	// Set auth-specific fields in tagged union
	// Note: For auth_method == 0 (System MSI), the union is not accessed
	if config.AuthMethod == 1 {
		C.geneva_set_cert(&cConfig, cCertPath, cCertPassword)
	} else if config.AuthMethod == 2 {
		C.geneva_set_workload_identity(&cConfig, cWorkloadIdentityResource)
	}
	// For auth_method 0 (System MSI), no union field needs to be set

	// Call Rust FFI to create client with error message buffer
	var handle *C.GenevaClientHandle
	errBuf := make([]byte, 1024) // Buffer for detailed error messages
	rc := C.geneva_client_new(
		&cConfig,
		&handle,
		(*C.char)(unsafe.Pointer(&errBuf[0])),
		C.size_t(len(errBuf)),
	)
	if rc != C.GENEVA_SUCCESS {
		// Try to extract detailed error message from buffer
		errMsg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", mapGenevaError(rc), errMsg)
		}
		return nil, mapGenevaError(rc)
	}

	client := &GenevaClient{handle: handle}

	// Set finalizer to ensure cleanup
	runtime.SetFinalizer(client, (*GenevaClient).Close)

	return client, nil
}

// UploadLogsSync uploads log data to Geneva synchronously (blocking)
func (c *GenevaClient) UploadLogsSync(data []byte) error {
	if c.handle == nil {
		return errors.New("geneva client is closed")
	}
	if len(data) == 0 {
		return errors.New("empty log data")
	}

	var batches *C.EncodedBatchesHandle
	errBuf := make([]byte, 1024) // Buffer for error messages
	rc := C.geneva_encode_and_compress_logs(
		c.handle,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&batches,
		(*C.char)(unsafe.Pointer(&errBuf[0])),
		C.size_t(len(errBuf)),
	)
	if rc != C.GENEVA_SUCCESS {
		errMsg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		if errMsg != "" {
			return fmt.Errorf("%w: %s", mapGenevaError(rc), errMsg)
		}
		return mapGenevaError(rc)
	}
	defer C.geneva_batches_free(batches)

	n := int(C.geneva_batches_len(batches))
	// Reuse the errBuf for upload errors
	for i := range n {
		res := C.geneva_upload_batch_sync(
			c.handle,
			batches,
			C.size_t(i),
			(*C.char)(unsafe.Pointer(&errBuf[0])),
			C.size_t(len(errBuf)),
		)
		if res != C.GENEVA_SUCCESS {
			// Extract error message from buffer
			errMsg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
			if errMsg != "" {
				return fmt.Errorf("failed to upload spans batch to Geneva Warm: %s", errMsg)
			}
			return mapGenevaError(res)
		}
	}
	return nil
}

// EncodedBatches wraps batches handle from FFI.
type EncodedBatches struct {
	handle *C.EncodedBatchesHandle
}

// Len returns number of batches.
func (b *EncodedBatches) Len() int {
	if b == nil || b.handle == nil {
		return 0
	}
	return int(C.geneva_batches_len(b.handle))
}

// Close frees the underlying batches handle.
func (b *EncodedBatches) Close() {
	if b != nil && b.handle != nil {
		C.geneva_batches_free(b.handle)
		b.handle = nil
	}
}

// EncodeAndCompressLogs uses FFI to create compressed batches for upload.
func (c *GenevaClient) EncodeAndCompressLogs(data []byte) (*EncodedBatches, error) {
	if c.handle == nil {
		return nil, errors.New("geneva client is closed")
	}
	if len(data) == 0 {
		return nil, errors.New("empty log data")
	}
	var batches *C.EncodedBatchesHandle
	errBuf := make([]byte, 1024)
	rc := C.geneva_encode_and_compress_logs(
		c.handle,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&batches,
		(*C.char)(unsafe.Pointer(&errBuf[0])),
		C.size_t(len(errBuf)),
	)
	if rc != C.GENEVA_SUCCESS {
		errMsg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", mapGenevaError(rc), errMsg)
		}
		return nil, mapGenevaError(rc)
	}
	return &EncodedBatches{handle: batches}, nil
}

// EncodeAndCompressSpans uses FFI to create compressed span batches for upload.
func (c *GenevaClient) EncodeAndCompressSpans(data []byte) (*EncodedBatches, error) {
	if c.handle == nil {
		return nil, errors.New("geneva client is closed")
	}
	if len(data) == 0 {
		return nil, errors.New("empty span data")
	}
	var batches *C.EncodedBatchesHandle
	errBuf := make([]byte, 1024)
	rc := C.geneva_encode_and_compress_spans(
		c.handle,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&batches,
		(*C.char)(unsafe.Pointer(&errBuf[0])),
		C.size_t(len(errBuf)),
	)
	if rc != C.GENEVA_SUCCESS {
		errMsg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", mapGenevaError(rc), errMsg)
		}
		return nil, mapGenevaError(rc)
	}
	return &EncodedBatches{handle: batches}, nil
}

// UploadBatch uploads a single batch index synchronously.
func (c *GenevaClient) UploadBatch(b *EncodedBatches, idx int) error {
	if c.handle == nil {
		return errors.New("geneva client is closed")
	}
	if b == nil || b.handle == nil {
		return errors.New("nil batches")
	}
	errBuf := make([]byte, 1024)
	res := C.geneva_upload_batch_sync(
		c.handle,
		b.handle,
		C.size_t(idx),
		(*C.char)(unsafe.Pointer(&errBuf[0])),
		C.size_t(len(errBuf)),
	)
	if res != C.GENEVA_SUCCESS {
		errMsg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		log.Printf("DEBUG: Upload failed with error message: %s", errMsg)
		if errMsg != "" {
			return fmt.Errorf("geneva upload failed: %s", errMsg)
		}
		return mapGenevaError(res)
	}
    //log.Printf("DEBUG: Upload successful for batch %d", idx)
	return nil
}

// UploadLogs uploads log data to Geneva (synchronous)
func (c *GenevaClient) UploadLogs(data []byte) error {
	return c.UploadLogsSync(data)
}

// mapGenevaError converts a C GenevaError to a Go error
func mapGenevaError(result C.GenevaError) error {
	switch result {
	case C.GENEVA_SUCCESS:
		return nil
	case C.GENEVA_INVALID_CONFIG:
		return errors.New("invalid geneva configuration")
	case C.GENEVA_INITIALIZATION_FAILED:
		return errors.New("geneva client initialization failed")
	case C.GENEVA_UPLOAD_FAILED:
		return errors.New("geneva upload failed")
	case C.GENEVA_INVALID_DATA:
		return errors.New("invalid log data")
	case C.GENEVA_INTERNAL_ERROR:
		return errors.New("geneva internal error")

		// Granular errors
	case genevaErrNullPointer:
		return errors.New("null pointer")
	case genevaErrEmptyInput:
		return errors.New("empty input")
	case genevaErrDecodeFailed:
		return errors.New("decode failed")
	case genevaErrIndexOutOfRange:
		return errors.New("index out of range")
	case genevaErrInvalidAuthMethod:
		return errors.New("invalid auth method")
	case genevaErrInvalidCertConfig:
		return errors.New("invalid certificate config")
	case genevaErrMissingEndpoint:
		return errors.New("missing endpoint")
	case genevaErrMissingEnvironment:
		return errors.New("missing environment")
	case genevaErrMissingAccount:
		return errors.New("missing account")
	case genevaErrMissingNamespace:
		return errors.New("missing namespace")
	case genevaErrMissingRegion:
		return errors.New("missing region")
	case genevaErrMissingTenant:
		return errors.New("missing tenant")
	case genevaErrMissingRoleName:
		return errors.New("missing role name")
	case genevaErrMissingRoleInstance:
		return errors.New("missing role instance")

	default:
		return fmt.Errorf("unknown Geneva error: %d", int(result))
	}
}

// Close frees the Geneva client resources
func (c *GenevaClient) Close() {
	if c.handle != nil {
		C.geneva_client_free(c.handle)
		c.handle = nil
		runtime.SetFinalizer(c, nil)
	}
}
