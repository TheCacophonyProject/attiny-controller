#!/bin/bash
systemctl daemon-reload
systemctl enable attiny-controller.service
systemctl restart attiny-controller.service