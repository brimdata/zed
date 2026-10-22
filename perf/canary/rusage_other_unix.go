//go:build unix && !darwin

package main

// On Linux and the BSDs, getrusage reports ru_maxrss in kilobytes.
const maxrssUnit = 1024
