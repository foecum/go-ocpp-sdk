package security

import (
	"encoding/base64"
	"fmt"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// CertificateHashData identifies an installed certificate by the hashes of its
// issuer and its serial number, as in OCSP (RFC 6960). It is used by
// DeleteCertificate and GetInstalledCertificateIds.
type CertificateHashData struct {
	// HashAlgorithm is the algorithm the hashes were made with.
	HashAlgorithm HashAlgorithm `json:"hashAlgorithm" ocpp:"required"`
	// IssuerNameHash is the hash of the DER encoded issuer name, as hexadecimal
	// (two digits per byte, at most 128 characters).
	IssuerNameHash string `json:"issuerNameHash" ocpp:"required"`
	// IssuerKeyHash is the hash of the issuer's DER encoded public key, as
	// hexadecimal (two digits per byte, at most 128 characters).
	IssuerKeyHash string `json:"issuerKeyHash" ocpp:"required"`
	// SerialNumber is the certificate's serial number as a hexadecimal string
	// without the 0x prefix, for example "FFF" for 4095 (at most 40 characters).
	SerialNumber string `json:"serialNumber" ocpp:"required"`
}

// Validate checks the structure.
func (h CertificateHashData) Validate() error {
	return types.First(
		types.CheckEnum("hashAlgorithm", h.HashAlgorithm, true),
		checkHex("issuerNameHash", h.IssuerNameHash, 128, true),
		checkHex("issuerKeyHash", h.IssuerKeyHash, 128, true),
		checkHex("serialNumber", h.SerialNumber, types.CiString40, false),
	)
}

// checkHex validates a required hexadecimal string of at most max characters.
// With wholeBytes set it must also have an even number of digits.
func checkHex(field, v string, max int, wholeBytes bool) error {
	if err := types.CheckString(field, v, max, true); err != nil {
		return err
	}
	for _, r := range v {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return ocpp.Invalid(field, "must be hexadecimal")
		}
	}
	if wholeBytes && len(v)%2 != 0 {
		return ocpp.Invalid(field, "must have two hexadecimal digits per byte")
	}
	return nil
}

// LogParameters says which log to retrieve and where to send it.
type LogParameters struct {
	// RemoteLocation is the URI to upload the log to (at most 512 characters).
	RemoteLocation string `json:"remoteLocation" ocpp:"required"`
	// OldestTimestamp is the oldest log entry to include.
	OldestTimestamp *types.DateTime `json:"oldestTimestamp,omitempty"`
	// LatestTimestamp is the latest log entry to include.
	LatestTimestamp *types.DateTime `json:"latestTimestamp,omitempty"`
}

// Validate checks the structure.
func (p LogParameters) Validate() error {
	if err := types.CheckURI("remoteLocation", p.RemoteLocation, true); err != nil {
		return err
	}
	return types.CheckString("remoteLocation", p.RemoteLocation, 512, true)
}

// Firmware describes the signed firmware of a SignedUpdateFirmware request.
type Firmware struct {
	// Location is the URI the firmware is retrieved from (at most 512 characters).
	Location string `json:"location" ocpp:"required"`
	// RetrieveDateTime is when the firmware shall be retrieved.
	RetrieveDateTime types.DateTime `json:"retrieveDateTime" ocpp:"required"`
	// InstallDateTime is when the firmware shall be installed.
	InstallDateTime *types.DateTime `json:"installDateTime,omitempty"`
	// SigningCertificate is the PEM encoded X.509 certificate the firmware was
	// signed with (at most 5500 characters).
	SigningCertificate string `json:"signingCertificate" ocpp:"required"`
	// Signature is the Base64 encoded firmware signature (at most 800 characters).
	Signature string `json:"signature" ocpp:"required"`
}

// Validate checks the structure.
func (f Firmware) Validate() error {
	if err := types.First(
		types.CheckURI("location", f.Location, true),
		types.CheckString("location", f.Location, 512, true),
		types.CheckDateTime("retrieveDateTime", f.RetrieveDateTime),
		checkPEM("signingCertificate", f.SigningCertificate, 5500, 1, pemCertificate),
		types.CheckString("signature", f.Signature, 800, true),
	); err != nil {
		return err
	}
	if _, err := base64.StdEncoding.DecodeString(f.Signature); err != nil {
		return ocpp.Invalid("signature", fmt.Sprintf("must be Base64 encoded: %v", err))
	}
	return nil
}
