// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func Raw(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func String(data []byte) string {
	hv := Raw(data)
	return hex.EncodeToString(hv[:])
}
