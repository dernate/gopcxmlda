package gopcxmlda

import "log"

func logError(err error, function string) {
	if err != nil {
		log.Printf("gopcxmlda error at %s: %s", function, err)
	}
}
