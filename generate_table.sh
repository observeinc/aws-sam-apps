#!/bin/bash

# Fetch available AWS regions
regions=$(aws ec2 describe-regions --query "Regions[].RegionName" --output text | tr '\t' '\n')

# Initialize the table with headers
echo "| Collection | Config | Forwarder |"
echo "|------------|--------|-----------|"

# Generate rows for each region
for region in $regions; do
    # Normalize region name for badge (replace '-' with '_')
    normalized_region=$(echo $region | tr '-' '_')
    
    # Create shield badge markdown
    badge_md="![Static Badge](https://img.shields.io/badge/$normalized_region-latest-blue?logo=amazonaws)"
    
    # Create CloudFormation console links
    collection_link="https://$region.console.aws.amazon.com/cloudformation/home?region=$region#/stacks/create/review?templateURL=https://observeinc-$region.s3.amazonaws.com/apps/collection/latest/packaged.yaml"
    config_link="https://$region.console.aws.amazon.com/cloudformation/home?region=$region#/stacks/create/review?templateURL=https://observeinc-$region.s3.amazonaws.com/apps/config/latest/packaged.yaml"
    forwarder_link="https://$region.console.aws.amazon.com/cloudformation/home?region=$region#/stacks/create/review?templateURL=https://observeinc-$region.s3.amazonaws.com/apps/forwarder/latest/packaged.yaml"
    
    # Generate table row with shield badges linking to CloudFormation console
    echo "| [$badge_md]($collection_link) | [$badge_md]($config_link) | [$badge_md]($forwarder_link) |"
done
