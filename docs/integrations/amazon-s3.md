---
sidebar_position: 1
sidebar_label: Amazon S3
---

# Amazon S3

Zed tools can access [Amazon S3](https://aws.amazon.com/s3/) and
S3-compatible storage via `s3://` URIs. Details are described below.

## Credentials

The URI parameters that are accepted by Zed components can be `s3://`
destinations that point to S3 storage. As most S3 buckets are non-public, this
requires the Zed tooling to have credentials for access. These credentials can
be provided in one of two ways.

1. If you have the [AWS CLI](https://aws.amazon.com/cli/) tools installed on
the same system where you're running Zed and have successfully completed
`aws configure`, the Zed tools will automatically find and use these saved
credentials and config in their well-known location. If you can successfully
execute operations such as `aws s3 ls` or `aws s3 cp` against an S3 URI, it is
expected that the Zed tools will be able to successfully make use of the same
S3 URI.

2. If the AWS CLI tools are absent or unconfigured, the necessary credentials
can be set in environment variables `AWS_ACCESS_KEY_ID`,
`AWS_SECRET_ACCESS_KEY`, and `AWS_REGION`. The values for these are the same as
are typically input in the first three settings entered during `aws configure`.

   ```
   $ aws configure
   AWS Access Key ID [********************]: 
   AWS Secret Access Key [********************]: 
   Default region name [*********]: 
   ```

## Endpoint

To use S3-compatible storage not provided by AWS, set the `AWS_S3_ENDPOINT`
environment variable to the hostname or URI of the provider.

## Wildcard Support

[Like the AWS CLI tools themselves](https://aws.amazon.com/premiumsupport/knowledge-center/s3-event-notification-filter-wildcard/#:~:text=Because%20the%20wildcard%20asterisk%20character,suffix%20object%20key%20name%20filter.),
Zed does not currently expand UNIX-style `*` wildcards in S3 URIs. If you
find this limitation is impacting your workflow, please add your use case
details as a comment in issue [zed/1994](https://github.com/brimdata/zed/issues/1994)
to help us track the priority of possible enhancements in this area.
