# Observe AWS Config Subscription Setup

The Observe AWS Config subscription stack attempts to forward data for an existing AWS Config installation.

## Overview

This stack installs EventBridge rules which process events from AWS Config:

- generates "copy" commands for events that denote a successful delivery of data to an S3 bucket
- forward change notification events 


## Template Configuration

### Parameters

The stack supports the following parameters:

| Parameter       | Type    | Description |
|-----------------|---------|-------------|
| **`TargetArn`** | String | Where to forward EventBridge events. |

### Outputs

No outputs
