#!/usr/bin/env sh
systemctl --system daemon-reload
systemctl stop lumeond.service
systemctl disable lumeond.service
