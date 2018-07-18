package main

import (
	"bytes"
	"html/template"
)

// ParseTemplate returns parsed template with data in string format
// If there is an error, it will return response with error data
func ParseTemplate(templateFileName string, data interface{}) (content string, err error) {

	tmpl, err := template.ParseFiles(templateFileName)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
