# Argos Discovery Dogfood Round 8 Prep

Date: 2026-05-02
Workspace root: `/tmp/argos-dogfood-20260502-round8-CCdPmM`
Full fixture workspace: `/tmp/argos-dogfood-20260502-round8-CCdPmM/full`
Empty fixture workspace: `/tmp/argos-dogfood-20260502-round8-CCdPmM/empty`
Binary: `/tmp/argos-dogfood-20260502-round8-CCdPmM/argos`

## Goal

Run the first dogfood round using the productized local harness:

- generate runner packets with neutral public case handles;
- keep oracle support labels, oracle IDs, source files, index tables, and raw
  knowledge files out of runner context;
- save structured reports under the round workspace;
- evaluate each saved report with `argos dogfood evaluate`.

This prep record intentionally does not include golden case IDs or expected
values. It records only the public runner packet set and coordinator commands.

## Prepared Packets

| Case | Operation | Fixture | Packet |
| --- | --- | --- | --- |
| `case-001` | list | full | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-001.md` |
| `case-002` | list-empty | empty | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-002.md` |
| `case-009` | find | full | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-009.md` |
| `case-017` | workflow | full | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-017.md` |
| `case-020` | cli-read | full | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-020.md` |
| `case-021` | cli-cite | full | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-021.md` |
| `case-023` | adapter | full | `/tmp/argos-dogfood-20260502-round8-CCdPmM/packets/case-023.md` |

## Coordinator Commands

The fixture workspaces were copied and indexed before packet generation.

```bash
/tmp/argos-dogfood-20260502-round8-CCdPmM/argos dogfood cases --json
/tmp/argos-dogfood-20260502-round8-CCdPmM/argos dogfood packet --case <case-handle> --workspace <fixture> --argos-binary /tmp/argos-dogfood-20260502-round8-CCdPmM/argos
```

After each fresh runner session saves a report:

```bash
/tmp/argos-dogfood-20260502-round8-CCdPmM/argos dogfood evaluate --case <case-handle> --report /tmp/argos-dogfood-20260502-round8-CCdPmM/reports/<case-handle>.md --json
```

## Runner Instructions

For each selected case, start a fresh runner context and provide only the
matching packet file. The runner should save a report using:

`docs/superpowers/templates/argos-discovery-dogfood-report.md`

Do not provide this prep report, previous dogfood reports, golden expectations,
or design history to the runner.

## Current Status

Superseded by the completed round report:

`docs/superpowers/reports/2026-05-02-argos-discovery-dogfood-round-8.md`
