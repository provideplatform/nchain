#!/bin/bash

pkill nchain_api || true
pkill nchain_consumer || true
pkill nchain_reachabilitydaemon || true
pkill nchain_statsdaemon || true
