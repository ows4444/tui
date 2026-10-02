// Package isolation holds the module-wide copy-isolation sweep: for every
// stateful widget Model it snapshots a value copy, drives Update on the
// original, and asserts the snapshot did not change. It has no non-test code.
package isolation
