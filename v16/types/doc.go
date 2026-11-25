// Package types holds the data types shared by the OCPP 1.6 profiles: the
// DateTime type, the enumerations, and the structures that appear in more than
// one message (IDTagInfo, MeterValue, SampledValue, ChargingProfile and so on).
//
// Conventions:
//
//   - Fields are named in Go style (ConnectorID, IDTag) and carry the exact JSON
//     names of the specification in their tags.
//   - Optional strings and enumerations use their zero value (empty) for
//     "absent" and are omitted from the JSON. Optional numbers and timestamps
//     are pointers, because zero is a meaningful value for them.
//   - A required field is marked `ocpp:"required"`: when decoding, the registry
//     rejects a payload that lacks it (see package ocppj). Validate then checks
//     lengths, ranges and enumeration values.
//   - Every enumeration has an IsValid method.
//
// Strings are checked for length only. The specification restricts them to
// printable ASCII, but real charge points send other characters often enough
// that rejecting them would do more harm than good.
package types
