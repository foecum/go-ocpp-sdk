// Package firmware implements the OCPP 1.6 Firmware Management profile:
// GetDiagnostics and UpdateFirmware (sent by the Central System), and
// DiagnosticsStatusNotification and FirmwareStatusNotification (sent by the
// Charge Point).
//
// Messages follow the conventions of package core: a Request and a Confirmation
// type per message, Validate methods, and Features for registering them.
package firmware
