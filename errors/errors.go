// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

// Package errors provides convenience functions for Kivik drivers to report
// meaningful errors. This package is not conisidered part of the kivik public
// API and is subject to change without notice.
package errors // import "github.com/go-kivik/kivik/v4/errors"

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// statusError is an error message bundled with an HTTP status code.
type statusError struct {
	statusCode int
	message    string
}

// MarshalJSON satisifies the json.Marshaler interface for the statusError
// type.
func (se *statusError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"error":  statusText(se.statusCode),
		"reason": se.message,
	})
}

func (se *statusError) Error() string {
	return se.message
}

// HTTPStatus returns the statusError's embedded HTTP status code.
func (se *statusError) HTTPStatus() int {
	return se.statusCode
}

// Reason returns the error's underlying reason.
func (se *statusError) Reason() string {
	return se.message
}

// New is a wrapper around the standard errors.New, to avoid the need for
// multiple imports.
func New(msg string) error {
	return errors.New(msg)
}

// Status returns a new error with the designated HTTP status.
func Status(status int, msg string) error {
	return &statusError{
		statusCode: status,
		message:    msg,
	}
}

// Statusf returns a new error with the designated HTTP status.
func Statusf(status int, format string, args ...interface{}) error {
	return &statusError{
		statusCode: status,
		message:    fmt.Sprintf(format, args...),
	}
}

type wrappedError struct {
	err        error
	statusCode int
}

func (e *wrappedError) Error() string {
	return e.err.Error()
}

func (e *wrappedError) HTTPStatus() int {
	return e.statusCode
}

func (e *wrappedError) Cause() error {
	return e.err
}

// WrapStatus bundles an existing error with a status code.
func WrapStatus(status int, err error) error {
	if err == nil {
		return nil
	}
	return &wrappedError{
		err:        err,
		statusCode: status,
	}
}

// Wrap is a wrapper around pkg/errors.Wrap()
func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

// Wrapf is a wrapper around pkg/errors.Wrapf()
func Wrapf(err error, format string, args ...interface{}) error {
	return errors.Wrapf(err, format, args...)
}

// Errorf is a wrapper around pkg/errors.Errorf()
func Errorf(format string, args ...interface{}) error {
	return errors.Errorf(format, args...)
}
