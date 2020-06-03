package utils

import "log"

func ReportError(label string, f func() error) {
	e := f()
	if e != nil {
		log.Printf("%v: Error: %v", label, e)
	}
}
