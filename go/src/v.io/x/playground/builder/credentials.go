// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Functions to create and bless principals.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
)

type credentials struct {
	Name     string
	Blesser  string
	Duration string
	Files    []string
}

const (
	credentialsDir     = "credentials" // Parent directory of all created credentials.
	identityProvider   = "playground"  // Blessing name of the "identity provider" that blesses all other principals
	defaultCredentials = "default"     // What codeFile.credentials defaults to if empty
)

var reservedCredentials = []string{identityProvider, "mounttabled", "proxyd", defaultCredentials}

func (c credentials) create() error {
	if err := c.init(); err != nil {
		return err
	}
	if c.Blesser == "" && c.Duration != "" {
		return c.initWithDuration()
	}
	if c.Blesser != "" {
		return c.getblessed()
	}
	return nil
}

func (c credentials) init() error {
	dir := path.Join(credentialsDir, c.Name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return c.toolCmd("", "create", dir, c.Name).Run()
	}
	return nil
}

func (c credentials) initWithDuration() error {
	// (1) principal blessself --for=<duration> <c.Name> | principal set default -
	// (2) principal get default | principal set forpeer - ...
	if err := c.pipe(c.toolCmd(c.Name, "blessself", "--for", c.Duration),
		c.toolCmd(c.Name, "set", "default", "-")); err != nil {
		return err
	}
	if err := c.pipe(c.toolCmd(c.Name, "get", "default"),
		c.toolCmd(c.Name, "set", "forpeer", "-", "...")); err != nil {
		return err
	}
	return nil
}

func (c credentials) getblessed() error {
	// (1) V23_CREDENTIALS=<c.Blesser> principal bless <c.Name> --for=<c.Duration> <c.Name> | V23_CREDENTIALS=<c.Name> principal set default -
	// (2) principal get default | principal set forpeer - ...
	duration := c.Duration
	if duration == "" {
		duration = "1h"
	}
	if err := c.pipe(c.toolCmd(c.Blesser, "bless", "--for", duration, path.Join(credentialsDir, c.Name), c.Name),
		c.toolCmd(c.Name, "set", "default", "-")); err != nil {
		return err
	}
	if err := c.pipe(c.toolCmd(c.Name, "get", "default"),
		c.toolCmd(c.Name, "set", "forpeer", "-", "...")); err != nil {
		return err
	}
	return nil
}

func (c credentials) pipe(from, to *exec.Cmd) error {
	buf := new(bytes.Buffer)
	from.Stdout = buf
	to.Stdin = buf
	if err := from.Run(); err != nil {
		return fmt.Errorf("%v %v: %v", from.Path, from.Args, err)
	}
	if err := to.Run(); err != nil {
		return fmt.Errorf("%v %v: %v", to.Path, to.Args, err)
	}
	return nil
}

func (c credentials) toolCmd(credentials string, args ...string) *exec.Cmd {
	cmd := makeCmd("<principal>", false, credentials, "principal", args...)
	// Set Stdout to /dev/null so that output does not leak into the
	// playground output. If the output is needed, it can be overridden by
	// clients of this method.
	cmd.Stdout = nil
	return cmd
}

func createCredentials(creds []credentials) error {
	debug("Generating credentials")
	for _, c := range creds {
		if err := c.create(); err != nil {
			return err
		}
	}
	return nil
}

func baseCredentials() []credentials {
	ret := []credentials{{Name: identityProvider}}
	for _, name := range reservedCredentials {
		if name != identityProvider {
			ret = append(ret, credentials{Name: name, Blesser: identityProvider})
		}
	}
	return ret
}

func rootCredentialsAtIdentityProvider(in []credentials) []credentials {
	out := make([]credentials, len(in))
	for idx, creds := range in {
		if creds.Blesser == "" {
			creds.Blesser = identityProvider
		}
		out[idx] = creds
	}
	return out
}

func isReservedCredential(name string) bool {
	for _, c := range reservedCredentials {
		if name == c {
			return true
		}
	}
	return false
}