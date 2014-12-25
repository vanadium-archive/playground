// Functions to create and bless principals.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"

	"v.io/veyron/veyron/lib/flags/consts"
)

type credentials struct {
	Name     string
	Blesser  string
	Duration string
	Files    []string
}

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
	dir := path.Join("credentials", c.Name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return c.toolCmd("", "create", dir, c.Name).Run()
	}
	return nil
}

func (c credentials) initWithDuration() error {
	// (1) principal blessself --for=<duration> <c.Name> | principal store setdefault -
	// (2) principal store default | principal store set - ...
	if err := c.pipe(c.toolCmd(c.Name, "blessself", "--for", c.Duration),
		c.toolCmd(c.Name, "store", "setdefault", "-")); err != nil {
		return err
	}
	if err := c.pipe(c.toolCmd(c.Name, "store", "default"),
		c.toolCmd(c.Name, "store", "set", "-", "...")); err != nil {
		return err
	}
	return nil
}

func (c credentials) getblessed() error {
	// (1) VEYRON_CREDENTIALS=<c.Blesser> principal bless <c.Name> --for=<c.Duration> <c.Name> | VEYRON_CREDENTIALS=<c.Name> principal store setdefault -
	// (2) principal store default | principal store set - ...
	duration := c.Duration
	if duration == "" {
		duration = "1h"
	}
	if err := c.pipe(c.toolCmd(c.Blesser, "bless", "--for", duration, path.Join("credentials", c.Name), c.Name),
		c.toolCmd(c.Name, "store", "setdefault", "-")); err != nil {
		return err
	}
	if err := c.pipe(c.toolCmd(c.Name, "store", "default"),
		c.toolCmd(c.Name, "store", "set", "-", "...")); err != nil {
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

func (c credentials) toolCmd(as string, args ...string) *exec.Cmd {
	cmd := makeCmd("", false, "principal", args...)
	if as != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%s", consts.VeyronCredentials, path.Join("credentials", as)))
	}
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
