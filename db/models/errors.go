package models

import (
	"fmt"
	"strings"
)

type DatabaseAccessError struct {
	msg string
}

func (e *DatabaseAccessError) Error() string { return e.msg }

func NewDatabaseAccessError(msg string) error {
	return &DatabaseAccessError{msg}
}

type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string { return e.msg }

func NewValidationError(msg string, values ...interface{}) error {
	return &ValidationError{msg: fmt.Sprintf(msg, values...)}
}

type JSONAPIErrors struct {
	Errors []JSONAPIError `json:"errors"`
}

type JSONAPIError struct {
	Detail string `json:"detail"`
}

func NewJSONAPIErrors() *JSONAPIErrors {
	fe := JSONAPIErrors{
		Errors: []JSONAPIError{},
	}
	return &fe
}

func NewJSONAPIErrorsWith(detail string) *JSONAPIErrors {
	fe := NewJSONAPIErrors()
	fe.Errors = append(fe.Errors, JSONAPIError{Detail: detail})
	return fe
}

func (jae *JSONAPIErrors) Error() string {
	var messages []string
	for _, e := range jae.Errors {
		messages = append(messages, e.Detail)
	}
	return strings.Join(messages, ",")
}

func (jae *JSONAPIErrors) Add(detail string) {
	jae.Errors = append(jae.Errors, JSONAPIError{Detail: detail})
}

func (jae *JSONAPIErrors) Merge(e error) {
	switch typed := e.(type) {
	case *JSONAPIErrors:
		jae.Errors = append(jae.Errors, typed.Errors...)
	default:
		jae.Add(e.Error())
	}
}

func (jae *JSONAPIErrors) CoerceEmptyToNil() error {
	if len(jae.Errors) == 0 {
		return nil
	}
	return jae
}
