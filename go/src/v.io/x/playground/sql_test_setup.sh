#!/bin/bash
# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Creates a playground_test database and playground_test user with access
# permissions to that database.

mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS playground_test; \
	GRANT ALL PRIVILEGES ON playground_test.* TO 'playground_test'@'localhost';"
