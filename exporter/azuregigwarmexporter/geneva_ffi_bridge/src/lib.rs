//! Geneva FFI Bridge for Go Integration
//!
//! This crate provides a simple bridge that re-exports the geneva-uploader-ffi
//! functionality from the registry package for CGO integration.

pub use geneva_uploader_ffi::*;

// Re-export all FFI functions and types for easy access from Go
// The geneva-uploader-ffi crate from the registry includes all necessary
// header files and FFI bindings
