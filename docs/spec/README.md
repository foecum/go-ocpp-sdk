# OCPP specifications

Reference material for the SDK. Published by the Open Charge Alliance (OCA) and
not covered by this repository's MIT licence: © Open Charge Alliance, made
available under the Creative Commons Attribution-NoDerivatives 4.0 International
licence (<https://creativecommons.org/licenses/by-nd/4.0/>). The files here are
unchanged copies (the schema zips are also extracted, file contents untouched);
do not edit them. The `go.mod` in this folder keeps them out of the Go module, so
`go get` does not download them. Download from
<https://openchargealliance.org/my-oca/ocpp/>. JSON (OCPP-J) only; the SDK does
not implement SOAP (OCPP-S).

## Precedence when documents disagree

1. Errata sheet
2. OCPP-J specification (framing and transport)
3. Main specification
4. JSON schema

If a schema and the spec text still conflict after errata, record it in
"Known gaps" in `AGENTS.md` and ask the owner.

## Expected files

### `1.6/` (OCPP 1.6 edition 2)

| File | Purpose |
|------|---------|
| `ocpp-1.6 edition 2.pdf` | Main specification |
| `ocpp-j-1.6-specification.pdf` | OCPP-J framing and transport |
| `ocpp-1.6-errata-sheet.pdf` | Errata for the main specification |
| `ocpp-j-1.6-errata-sheet.pdf` | Errata for OCPP-J |
| `OCPP 1.6 security whitepaper edition 3.pdf` | The security extension: certificate management, signed firmware, logs, security events and the security profiles |
| `schemas/json/*.json` | JSON schemas, one per request and response. The core and the security extension messages share this folder |
| `schemas/soap/*.wsdl` | SOAP WSDLs. Present in the OCA download, **unused** |

### `2.0.1/` (OCPP 2.0.1 edition 4)

| File | Purpose |
|------|---------|
| `OCPP-2.0.1_edition4_part0_introduction.pdf` | Part 0, introduction |
| `OCPP-2.0.1_edition4_part1_architecture_topology.pdf` | Part 1, architecture and topology |
| `OCPP-2.0.1_edition4_part2_specification.pdf` | Part 2, main specification |
| `OCPP-2.0.1_edition4_part2_appendices_v15.pdf` | Part 2 appendices |
| `OCPP-2.0.1_part3_JSON_schemas.zip` | Part 3, JSON schemas (extract to `schemas/`) |
| `OCPP-2.0.1_edition4_part4_ocpp-j-specification.pdf` | Part 4, OCPP-J |
| `OCPP-2.0.1_edition4_part5_certification_profiles.pdf` | Part 5, for checking behaviour only |
| `OCPP-2.0.1_edition4_part6-testcases.pdf` | Part 6, for checking behaviour only |
| `OCPP-2.0.1_edition4_errata_2026-09.pdf` | Errata for all parts |
| `Appendices_CSV_v1.5.zip` | Components and variables, machine readable (extract to `appendices-csv/`) |
| `ReadMe.md` | OCA release notes for this set |

### `2.1/` (OCPP 2.1 edition 2)

Same structure as 2.0.1, with `OCPP-2.1_*` file names,
`OCPP-2.1_edition2_part2_appendices_v21.pdf` and `Appendices_CSV_v2.1.zip`.

## Notes

- Extract the schemas zips to `schemas/` and the appendices zips to
  `Appendices_CSV_v1.5/` (2.0.1) and `Appendices_CSV_v2.1/` (2.1); the generator
  (`internal/gen`) reads them from there. The 2.1 zip holds one folder: move its files up.
- Prefer the appendices CSVs over copying component and variable tables out of
  the PDF.
- CALLERROR spelling differs: 1.6 uses `FormationViolation` and
  `OccurenceConstraintViolation` (sic); 2.0.1 uses `FormatViolation` and
  `OccurrenceConstraintViolation`.
