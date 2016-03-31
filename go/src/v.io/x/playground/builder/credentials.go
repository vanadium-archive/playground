// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Functions to create and bless principals.

package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"v.io/v23/security"
	libsecurity "v.io/x/ref/lib/security"
	"v.io/x/ref/services/agent"
	"v.io/x/ref/services/agent/agentlib"
	"v.io/x/ref/services/agent/keymgr"
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

var reservedCredentials = []string{identityProvider, "mounttabled", "xproxyd", defaultCredentials}

func isReservedCredential(name string) bool {
	for _, c := range reservedCredentials {
		if name == c {
			return true
		}
	}
	return false
}

type credentialsManager struct {
	sync.Mutex
	dir    string
	keyMgr agent.KeyManager
	// Map from blessing to socket file path.
	socks map[string]string
}

func (cm *credentialsManager) socket(name string) (string, error) {
	cm.Lock()
	defer cm.Unlock()
	return cm.socketLocked(name)
}

// called with cm's lock held.
func (cm *credentialsManager) socketLocked(name string) (string, error) {
	sock, ok := cm.socks[name]
	if !ok {
		return "", fmt.Errorf("principal for blessing \"%s\" doesn't exist", name)
	}
	return sock, nil
}

func (cm *credentialsManager) principal(name string) (agent.Principal, error) {
	cm.Lock()
	defer cm.Unlock()
	return cm.principalLocked(name)
}

// called with cm's lock held.
func (cm *credentialsManager) principalLocked(name string) (agent.Principal, error) {
	sock, err := cm.socketLocked(name)
	if err != nil {
		return nil, err
	}
	return agentlib.NewAgentPrincipal(sock, 0)
}

func (cm *credentialsManager) createPrincipal(blesser agent.Principal, name string, expiresAfter time.Duration) error {
	cm.Lock()
	defer cm.Unlock()
	if _, ok := cm.socks[name]; ok {
		return fmt.Errorf("principal with blessing name \"%s\" already exists", name)
	}
	handle, err := cm.keyMgr.NewPrincipal(true)
	if err != nil {
		return err
	}
	sockPath := filepath.Join(cm.dir, fmt.Sprintf("sock%d", len(cm.socks)))
	if err := cm.keyMgr.ServePrincipal(handle, sockPath); err != nil {
		return err
	}
	cm.socks[name] = sockPath
	p, err := cm.principalLocked(name)
	if err != nil {
		return err
	}
	defer p.Close()
	expiry, err := security.NewExpiryCaveat(time.Now().Add(expiresAfter))
	if err != nil {
		return err
	}
	var blessing security.Blessings
	if blesser == nil {
		// Self-blessed.
		if blessing, err = p.BlessSelf(name, expiry); err != nil {
			return err
		}
	} else {
		with, _ := blesser.BlessingStore().Default()
		if blessing, err = blesser.Bless(p.PublicKey(), with, name, expiry); err != nil {
			return err
		}
	}
	return libsecurity.SetDefaultBlessings(p, blessing)
}

func newCredentialsManager(creds []credentials) (*credentialsManager, error) {
	keyMgr, err := keymgr.NewLocalAgent(credentialsDir, nil)
	if err != nil {
		return nil, err
	}
	credsMgr := &credentialsManager{
		dir:    credentialsDir,
		keyMgr: keyMgr,
		socks:  make(map[string]string),
	}
	// Create the root identity provider.
	if err := credsMgr.createPrincipal(nil, identityProvider, time.Hour); err != nil {
		return nil, err
	}
	rootPrincipal, err := credsMgr.principal(identityProvider)
	if err != nil {
		return nil, err
	}
	defer rootPrincipal.Close()
	// Create the other reserved principals.
	for _, name := range reservedCredentials {
		if name != identityProvider {
			if err := credsMgr.createPrincipal(rootPrincipal, name, time.Hour); err != nil {
				return nil, err
			}
		}
	}
	// Create all the user-specified principals.
	for _, cred := range creds {
		var blesser agent.Principal
		if cred.Blesser == "" {
			blesser = rootPrincipal
		} else {
			blesser, err = credsMgr.principal(cred.Blesser)
			if err != nil {
				return nil, err
			}
			defer blesser.Close()
		}
		expiry := time.Hour
		if cred.Duration != "" {
			expiry, err = time.ParseDuration(cred.Duration)
			if err != nil {
				return nil, err
			}
		}
		if err := credsMgr.createPrincipal(blesser, cred.Name, expiry); err != nil {
			return nil, err
		}
	}

	return credsMgr, nil
}

func (cm *credentialsManager) Close() error {
	return cm.keyMgr.Close()
}
