#!/usr/bin/env sh
systemctl --system daemon-reload
systemctl enable lumeond.service
systemctl start lumeond.service
