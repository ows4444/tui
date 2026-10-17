package main

import (
	"io/fs"
	"testing/fstest"
)

// sampleDocs is a small made-up documentation set that links to itself, so the
// reader has pages to move between when it is given no folder.
func sampleDocs() fs.FS {
	f := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	return fstest.MapFS{
		"README.md": f(`# Parcel docs

Parcel moves small files between machines. Start with the [quick start](quickstart.md),
read about the [table widget](widgets/table.md), or see the [FAQ](faq.md).

## What is in here

- [Quick start](quickstart.md): send your first file
- [Table](widgets/table.md) and [Forms](widgets/forms.md): the two screens you will use most
- [Testing](testing.md): check a transfer without a network
- [FAQ](faq.md): the questions people ask first

The project lives at [example.com/parcel](https://example.com/parcel).
`),
		"quickstart.md": f(`# Quick start

Install Parcel, then send a file.

## Install

Download the archive for your system and put ` + "`parcel`" + ` on your path.

## Send a file

` + "```sh" + `
parcel send report.pdf alice@laptop
` + "```" + `

The receiver gets a prompt. If it does not, read [troubleshooting](faq.md#troubleshooting).

## Next steps

See the [table](widgets/table.md#columns) for how transfers are listed, and
[testing](testing.md) to try it offline.
`),
		"widgets/table.md": f(`# Table

The transfer table lists every file in flight.

## Columns

| Column | Meaning |
|--------|---------|
| Name   | the file being sent |
| Size   | its size on disk |
| Speed  | bytes a second, averaged over five seconds |

## Sorting

Click a column header to sort by it. Click again to reverse.

Back to the [forms](forms.md) or the [index](../README.md).
`),
		"widgets/forms.md": f(`# Forms

Forms ask for an address and a note before a transfer starts.

## Fields

The address must look like name@host. The note is optional.

## Validation

A wrong address is reported under the field. See the [table](table.md) for what happens next.
`),
		"testing.md": f(`# Testing

Parcel can run a transfer against a fake network.

## Offline mode

Set ` + "`PARCEL_OFFLINE=1`" + ` and every transfer succeeds instantly. Useful for scripts and for the
[quick start](quickstart.md).

## Golden files

Compare a transfer log with a saved copy to catch changes.
`),
		"faq.md": f(`# FAQ

## Does it need an account?

No. Parcel talks directly between two machines.

## Troubleshooting

If the receiver never gets a prompt, check that both sides can reach each other and that no
firewall blocks the port. Then try [offline mode](testing.md#offline-mode) to rule out the network.

## Where is the source?

On [GitHub](https://github.com/example/parcel).
`),
	}
}
