#!/usr/bin/env sh
if systemctl is-active --quiet lumeond.service
then
  systemctl stop lumeond.service
fi
