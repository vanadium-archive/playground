// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Utility functions for error handling.

package lib

import (
	"fmt"
)

// MergeErrors returns nil if both errors passed to it are nil.
// Otherwise, it concatenates non-nil error messages.
func MergeErrors(err1, err2 error, joiner string) error {
	if err1 == nil {
		return err2
	} else if err2 == nil {
		return err1
	} else {
		return fmt.Errorf("%v%s%v", err1, joiner, err2)
	}
}
