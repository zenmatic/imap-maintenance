#!/bin/bash

set -e
dovecot
tail -F /var/log/dovecot.log
