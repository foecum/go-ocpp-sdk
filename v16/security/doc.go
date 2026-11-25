// Package security implements the OCPP 1.6 security extension, defined in the
// "Improved security for OCPP 1.6-J" whitepaper (edition 3): certificate
// management, signed firmware updates, log retrieval and security event
// notifications.
//
// Messages sent by the Central System: CertificateSigned, DeleteCertificate,
// ExtendedTriggerMessage, GetInstalledCertificateIds, GetLog, InstallCertificate
// and SignedUpdateFirmware. Messages sent by the Charge Point:
// LogStatusNotification, SecurityEventNotification, SignCertificate and
// SignedFirmwareStatusNotification.
//
// The messages follow the conventions of package core: a Request and a
// Confirmation type per message, Validate methods and Features for registering
// them. Certificates and certificate requests are PEM encoded; Validate checks
// that they decode as PEM blocks of the right type, but does not parse the X.509
// contents.
//
// The extension also defines how the Basic-auth password (the AuthorizationKey)
// looks: see CheckAuthorizationKey and GenerateAuthorizationKey. The transport
// side of the security profiles is in package ws.
package security
