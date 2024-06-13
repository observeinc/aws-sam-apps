#!/bin/bash

# Fetch available AWS regions
regions=$(aws ec2 describe-regions --query "Regions[].RegionName" --output text | tr '\t' '\n')

# Initialize the table with headers
echo "| Stack | Config | Forwarder |"
echo "|------------|--------|-----------|"

# Generate rows for each region
for region in $regions; do
    # Normalize region name for badge (replace '-' with '_')
    normalized_region=$(echo $region | tr '-' '_')
    
    # Create shield badge markdown
    badge_md="![Static Badge](https://img.shields.io/badge/$normalized_region-latest-blue?logo=amazonaws)"
    
    # Create CloudFormation console links
    stack_link="https://$region.console.aws.amazon.com/cloudformation/home?region=$region#/stacks/create/review?templateURL=https://observeinc-$region.s3.amazonaws.com/aws-sam-apps/latest/stack.yaml"
    config_link="https://$region.console.aws.amazon.com/cloudformation/home?region=$region#/stacks/create/review?templateURL=https://observeinc-$region.s3.amazonaws.com/aws-sam-apps/latest/config.yaml"
    forwarder_link="https://$region.console.aws.amazon.com/cloudformation/home?region=$region#/stacks/create/review?templateURL=https://observeinc-$region.s3.amazonaws.com/aws-sam-apps/latest/forwarder.yaml"
    
    # Generate table row with shield badges linking to CloudFormation console
    echo "| [$badge_md]($stack_link) | [$badge_md]($config_link) | [$badge_md]($forwarder_link) |"
done
